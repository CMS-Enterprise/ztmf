package model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for ztmf#491: a system holds at most one answer per question per
// data call, enforced by migration 0061's unique index, kept derivable by
// 0060's functionid column, and honoured by Score.Save's natural-key resolve.

type uniqueAnswerFixture struct {
	dataCallID       int32
	fismaSystemID    int32
	functionID       int32
	optionA, optionB int32
	otherFunctionID  int32
	otherOption      int32
}

// newUniqueAnswerFixture builds an isolated cycle with two questions, the first
// having two selectable options so a second answer to the SAME question can be
// attempted. The data call carries the integrationTestPrefix so the sweep
// cascades its scores away, and a far-future deadline so Save's deadline check
// passes for a non-admin.
//
// Every lookup is fully ordered: a bare LIMIT 1 against the empire seed is
// plan-dependent. The second question is a control - it proves a guard is per
// question rather than collapsing a system's whole answer set.
func newUniqueAnswerFixture(t *testing.T, ctx context.Context, conn *pgxpool.Conn, label string) uniqueAnswerFixture {
	t.Helper()

	var f uniqueAnswerFixture
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%suniq_%s_%d", integrationTestPrefix, label, time.Now().UnixNano())).Scan(&f.dataCallID))

	rows, err := conn.Query(ctx, `
		SELECT fo.functionid, MIN(fo.functionoptionid), MAX(fo.functionoptionid)
		  FROM functionoptions fo
		 GROUP BY fo.functionid
		HAVING COUNT(*) >= 2
		 ORDER BY fo.functionid
		 LIMIT 2
	`)
	require.NoError(t, err)
	var picked [][3]int32
	for rows.Next() {
		var r [3]int32
		require.NoError(t, rows.Scan(&r[0], &r[1], &r[2]))
		picked = append(picked, r)
	}
	rows.Close()
	require.NoError(t, rows.Err())
	require.Len(t, picked, 2, "need two distinct functions with at least two functionoptions each")

	f.functionID, f.optionA, f.optionB = picked[0][0], picked[0][1], picked[0][2]
	f.otherFunctionID, f.otherOption = picked[1][0], picked[1][1]

	require.NoError(t, conn.QueryRow(ctx,
		`SELECT fismasystemid FROM fismasystems ORDER BY fismasystemid LIMIT 1`,
	).Scan(&f.fismaSystemID))

	return f
}

// empireOwnerCtx puts a fixture user in context so queryRow's recordEvent hook
// fires - without one it returns early and the event assertions below would
// pass vacuously. Empire fixtures only; never real CMS users.
func empireOwnerCtx(ctx context.Context) context.Context {
	return UserToContext(ctx, &User{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "Grand.Moff@DeathStar.Empire",
		FullName: "Grand Moff Tarkin",
		Role:     "OWNER",
	})
}

// lastScoreEventAction returns the most recent recorded action for a scoreid.
func lastScoreEventAction(t *testing.T, ctx context.Context, conn *pgxpool.Conn, scoreID int32) string {
	t.Helper()
	var action string
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT action FROM events
		 WHERE resource = 'public.scores' AND (payload->>'scoreid')::int = $1
		 ORDER BY createdat DESC, eventid DESC
		 LIMIT 1
	`, scoreID).Scan(&action))
	return action
}

// TestSaveCreateResolvesToExistingAnswerIntegration is the headline ztmf#491
// behaviour: a caller that does not know the existing scoreid must UPDATE the
// stored answer, not create a second row.
//
// The event assertion is the load-bearing one. An INSERT ... ON CONFLICT DO
// UPDATE would satisfy every other assertion here while recording action
// 'created' for a write that updated, because recordEvent derives the action
// from the squirrel builder's Go type.
func TestSaveCreateResolvesToExistingAnswerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "resolve")

	var existingID int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes, status)
		VALUES ($1, $2, $3, 'first answer', $4)
		RETURNING scoreid
	`, f.fismaSystemID, f.optionA, f.dataCallID, scoreStatusNotStarted).Scan(&existingID))

	notes := "answered without knowing the scoreid"
	saved, err := (&Score{
		FismaSystemID:    f.fismaSystemID,
		FunctionOptionID: f.optionB, // a different option of the SAME question
		DataCallID:       f.dataCallID,
		Notes:            &notes,
	}).Save(empireOwnerCtx(ctx))
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, existingID, saved.ScoreID, "the create must resolve onto the existing answer, not mint a new row")
	assert.EqualValues(t, f.optionB, saved.FunctionOptionID, "the new answer must be stored")
	assert.Equal(t, scoreStatusDone, saved.Status, "a real edit marks the answer done for this cycle")

	var rowsForQuestion int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM scores
		 WHERE fismasystemid = $1 AND datacallid = $2 AND functionid = $3
	`, f.fismaSystemID, f.dataCallID, f.functionID).Scan(&rowsForQuestion))
	assert.Equal(t, 1, rowsForQuestion, "exactly one answer per question per cycle")

	assert.Equal(t, eventActionUpdated, lastScoreEventAction(t, ctx, conn, existingID),
		"the write updated an existing answer and the audit log must say so, not 'created'")
}

// TestSaveCreateIsIdempotentOnRepeatIntegration pins that the ztmf#412/#413
// audit-preserving no-op still applies once a create can resolve onto an
// existing row. A repeated identical POST is a read-through, and must not
// restamp the editor or record a second event.
func TestSaveCreateIsIdempotentOnRepeatIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "idempotent")
	userCtx := empireOwnerCtx(ctx)

	notes := "same answer twice"
	mk := func() *Score {
		return &Score{
			FismaSystemID:    f.fismaSystemID,
			FunctionOptionID: f.optionA,
			DataCallID:       f.dataCallID,
			Notes:            &notes,
		}
	}

	first, err := mk().Save(userCtx)
	require.NoError(t, err)
	require.NotNil(t, first)
	eventsAfterFirst := countScoreEvents(t, ctx, conn, first.ScoreID)

	second, err := mk().Save(userCtx)
	require.NoError(t, err)
	require.NotNil(t, second)

	assert.Equal(t, first.ScoreID, second.ScoreID, "the repeat must land on the same row")
	assert.Equal(t, eventsAfterFirst, countScoreEvents(t, ctx, conn, first.ScoreID),
		"a read-through repeat records no event - the no-op guard must survive the natural-key resolve")
}

// TestSaveRejectsOptionFromAnotherQuestionIntegration covers the update path's
// half of the natural key. A score's question is part of its identity, so an
// update carrying an option from a DIFFERENT question is malformed.
//
// On main this request is how a duplicate gets manufactured: the row is
// rewritten to the new question, which both collides with that question's
// existing answer and throws away the answer this row was holding. Rejecting is
// also why the update path does not resolve the natural key the way the create
// path does - silently writing the other row would leave a buggy client
// believing it had edited the row it named.
func TestSaveRejectsOptionFromAnotherQuestionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "redirect")

	var targetID, otherID int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, 'the question being answered') RETURNING scoreid
	`, f.fismaSystemID, f.optionA, f.dataCallID).Scan(&targetID))
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, 'an unrelated question') RETURNING scoreid
	`, f.fismaSystemID, f.otherOption, f.dataCallID).Scan(&otherID))

	notes := "aimed at the wrong scoreid"
	saved, err := (&Score{
		ScoreID:          otherID, // names the OTHER question's row
		FismaSystemID:    f.fismaSystemID,
		FunctionOptionID: f.optionB, // but carries the FIRST question's option
		DataCallID:       f.dataCallID,
		Notes:            &notes,
	}).Save(empireOwnerCtx(ctx))

	require.Error(t, err, "an option from another question must be rejected, not written elsewhere")
	assert.Nil(t, saved)

	var invalid *InvalidInputError
	require.ErrorAs(t, err, &invalid, "must be a 400-shaped field error, not a 500")
	assert.Contains(t, invalid.Data(), "functionoptionid", "the offending field must be named")

	// Neither row moved: not the one that was named, and not the one the old
	// redirect behaviour would have written.
	var namedOption int32
	var namedNotes, targetNotes string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT functionoptionid, notes FROM scores WHERE scoreid = $1`, otherID).Scan(&namedOption, &namedNotes))
	assert.EqualValues(t, f.otherOption, namedOption, "the named row must keep its own question's answer")
	assert.Equal(t, "an unrelated question", namedNotes)

	require.NoError(t, conn.QueryRow(ctx,
		`SELECT notes FROM scores WHERE scoreid = $1`, targetID).Scan(&targetNotes))
	assert.Equal(t, "the question being answered", targetNotes,
		"a rejected update must not have written the natural-key row either")
}

// TestSaveUpdateAllowsSiblingOptionOfSameQuestionIntegration is the other half
// of the check above: changing the answer WITHIN a question is the ordinary edit
// and must keep working. Without this, a guard that rejected everything would
// pass the rejection test and break the questionnaire.
func TestSaveUpdateAllowsSiblingOptionOfSameQuestionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "sibling")

	var id int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, 'first pass') RETURNING scoreid
	`, f.fismaSystemID, f.optionA, f.dataCallID).Scan(&id))

	notes := "changed my mind"
	saved, err := (&Score{
		ScoreID:          id,
		FismaSystemID:    f.fismaSystemID,
		FunctionOptionID: f.optionB, // a sibling option of the SAME question
		DataCallID:       f.dataCallID,
		Notes:            &notes,
	}).Save(empireOwnerCtx(ctx))
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, id, saved.ScoreID)
	assert.EqualValues(t, f.optionB, saved.FunctionOptionID, "the answer must change within the question")
	assert.Equal(t, scoreStatusDone, saved.Status)
}

// TestSaveConcurrentCreatesResolveToOneRowIntegration covers the race the
// natural-key lookup alone cannot close: two writers both find no row, both
// attempt an INSERT, and the loser takes 23505. Save retries it once as the
// update it always was, so neither caller sees an error.
//
// Concurrent submission is one of the three ways ztmf#491 says duplicates were
// created, and with no cleanup migration in this PR the retry is the only thing
// covering it.
func TestSaveConcurrentCreatesResolveToOneRowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "race")
	userCtx := empireOwnerCtx(ctx)

	const writers = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0, writers)
	ids := make([]int32, 0, writers)

	start := make(chan struct{})
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			notes := fmt.Sprintf("writer %d", i)
			s := &Score{
				FismaSystemID:    f.fismaSystemID,
				FunctionOptionID: f.optionA,
				DataCallID:       f.dataCallID,
				Notes:            &notes,
			}
			<-start // release together, so the lookups genuinely interleave
			saved, err := s.Save(userCtx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			ids = append(ids, saved.ScoreID)
		}(i)
	}
	close(start)
	wg.Wait()

	assert.Empty(t, errs, "a writer that lost the natural-key race must be retried as an update, not surfaced as an error")
	require.NotEmpty(t, ids)
	for _, id := range ids {
		assert.Equal(t, ids[0], id, "every writer must converge on the same row")
	}

	var rowsForQuestion int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM scores
		 WHERE fismasystemid = $1 AND datacallid = $2 AND functionid = $3
	`, f.fismaSystemID, f.dataCallID, f.functionID).Scan(&rowsForQuestion))
	assert.Equal(t, 1, rowsForQuestion, "concurrent creates must leave exactly one answer")
}

// TestScoresUniqueAnswerConstraintIntegration asserts the guarantee at the
// database layer, independent of any Go code path - a bulk load or a psql
// session gets the same answer. Unlike the trigger and the composite FK, a
// unique index is still enforced under session_replication_role = 'replica',
// which is how the duplicates it prevents were created.
func TestScoresUniqueAnswerConstraintIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "constraint")

	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
	`, f.fismaSystemID, f.optionA, f.dataCallID)
	require.NoError(t, err)

	// A different option of the SAME question: distinct on functionoptionid, so
	// this only fails if the key is the question rather than the answer.
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
	`, f.fismaSystemID, f.optionB, f.dataCallID)
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
	assert.Equal(t, "scores_fismasystem_datacall_function_uniq", pgErr.ConstraintName)
	assert.ErrorIs(t, trapError(err), ErrNotUnique, "the model layer must surface this as a 400, not a 500")

	// The control question is unaffected - the key is per question.
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
	`, f.fismaSystemID, f.otherOption, f.dataCallID)
	assert.NoError(t, err, "a different question on the same system and cycle must still be insertable")
}

// TestScoresFunctionIDTriggerIntegration pins that scores.functionid is derived,
// never trusted. The unique index is only as good as this column.
func TestScoresFunctionIDTriggerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "trigger")

	readFunctionID := func(scoreID int32) int32 {
		var got int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT functionid FROM scores WHERE scoreid = $1`, scoreID).Scan(&got))
		return got
	}

	t.Run("DerivedOnInsert", func(t *testing.T) {
		var id int32
		require.NoError(t, conn.QueryRow(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
			RETURNING scoreid
		`, f.fismaSystemID, f.optionA, f.dataCallID).Scan(&id))
		assert.Equal(t, f.functionID, readFunctionID(id))
	})

	t.Run("SuppliedValueIsOverwritten", func(t *testing.T) {
		var id int32
		require.NoError(t, conn.QueryRow(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid, functionid)
			VALUES ($1, $2, $3, $4) RETURNING scoreid
		`, f.fismaSystemID, f.otherOption, f.dataCallID, f.functionID).Scan(&id))
		assert.Equal(t, f.otherFunctionID, readFunctionID(id),
			"a caller-supplied functionid must be replaced by the derived one, not stored")
	})

	t.Run("FollowsAnOptionChange", func(t *testing.T) {
		var id int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT scoreid FROM scores
			WHERE fismasystemid=$1 AND datacallid=$2 AND functionid=$3
		`, f.fismaSystemID, f.dataCallID, f.otherFunctionID).Scan(&id))

		_, err := conn.Exec(ctx, `UPDATE scores SET functionoptionid = $1 WHERE scoreid = $2`, f.optionB, id)
		require.Error(t, err, "moving this answer onto the already-answered question must hit the unique index")

		// Same move, but onto a question with no answer yet: functionid follows.
		var freeOption, freeFunction int32
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT fo.functionoptionid, fo.functionid FROM functionoptions fo
			 WHERE fo.functionid NOT IN ($1, $2)
			 ORDER BY fo.functionid, fo.functionoptionid LIMIT 1
		`, f.functionID, f.otherFunctionID).Scan(&freeOption, &freeFunction))

		_, err = conn.Exec(ctx, `UPDATE scores SET functionoptionid = $1 WHERE scoreid = $2`, freeOption, id)
		require.NoError(t, err)
		assert.Equal(t, freeFunction, readFunctionID(id), "functionid must follow the option to its new question")
	})

	t.Run("DirectWriteIsRestored", func(t *testing.T) {
		var id int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT scoreid FROM scores
			WHERE fismasystemid=$1 AND datacallid=$2 AND functionid=$3
		`, f.fismaSystemID, f.dataCallID, f.functionID).Scan(&id))

		_, err := conn.Exec(ctx, `UPDATE scores SET functionid = 987654 WHERE scoreid = $1`, id)
		require.NoError(t, err)
		assert.Equal(t, f.functionID, readFunctionID(id),
			"writing functionid directly must be corrected, not persisted - otherwise the unique key is forgeable")
	})

	t.Run("UnknownOptionIsAForeignKeyViolation", func(t *testing.T) {
		_, err := conn.Exec(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, 987654, $2)
		`, f.fismaSystemID, f.dataCallID)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		// 23503, not 23502. NOT NULL is checked at tuple insertion, before the
		// AFTER-trigger FK fires, and trapError has no 23502 case - so without the
		// trigger's explicit raise this would be a 500 where it is a 400 today.
		assert.Equal(t, "23503", pgErr.Code, "a bad functionoptionid must stay a 400, not become a 500")
		assert.ErrorIs(t, trapError(err), ErrNoReference)
	})
}

// TestScoresCompositeFunctionFKIntegration covers the one failure mode the
// trigger structurally cannot see: it fires on writes to scores, never on
// writes to functionoptions. That catalogue is edited by hand out of band, and
// repointing an option at a different function would silently stale every
// score referencing it.
func TestScoresCompositeFunctionFKIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	f := newUniqueAnswerFixture(t, ctx, conn, "catalogfk")

	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
	`, f.fismaSystemID, f.optionA, f.dataCallID)
	require.NoError(t, err)

	_, err = conn.Exec(ctx,
		`UPDATE functionoptions SET functionid = $1 WHERE functionoptionid = $2`,
		f.otherFunctionID, f.optionA)
	require.Error(t, err, "repointing a referenced functionoption must be blocked while a score depends on it")

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23503", pgErr.Code)
	assert.Equal(t, "scores_functionoption_function_fkey", pgErr.ConstraintName)
}

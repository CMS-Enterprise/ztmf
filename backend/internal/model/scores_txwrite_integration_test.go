package model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// txWriteFixture is one writable answer: a data call whose deadline is far
// enough out that a non-admin Save clears validateDeadline, plus a real
// (system, functionoption) pair. The data call carries integrationTestPrefix so
// purgeIntegrationTestRows cascades its scores away.
type txWriteFixture struct {
	dataCallID       int32
	fismaSystemID    int32
	functionOptionID int32
	editorCtx        context.Context
}

func newTxWriteFixture(t *testing.T, conn *pgxpool.Conn, name string) txWriteFixture {
	t.Helper()
	ctx := context.Background()

	var dc int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%s%s_%d", integrationTestPrefix, name, time.Now().UnixNano())).Scan(&dc))

	// Fully ordered rather than a bare LIMIT 1: against the empire seed an
	// unordered pick is plan-dependent and the fixture would drift between runs.
	var sys, opt int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fs.fismasystemid, fo.functionoptionid
		FROM fismasystems fs
		CROSS JOIN LATERAL (
			SELECT fo.functionoptionid
			FROM functionoptions fo
			INNER JOIN functions f ON f.functionid = fo.functionid
			ORDER BY fo.functionoptionid
			LIMIT 1
		) fo
		WHERE fs.decommissioned = FALSE
		ORDER BY fs.fismasystemid
		LIMIT 1
	`).Scan(&sys, &opt), "need one active system and one function option in the seed")

	var editorID, editorRole string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT userid, role FROM users ORDER BY (role = 'OWNER') DESC, userid LIMIT 1`,
	).Scan(&editorID, &editorRole), "need a seeded user to attribute the write to")

	return txWriteFixture{
		dataCallID:       dc,
		fismaSystemID:    sys,
		functionOptionID: opt,
		editorCtx:        UserToContext(ctx, &User{UserID: editorID, Role: editorRole}),
	}
}

// scoreEvents returns this score's audit events oldest-first, projecting the
// notes out of the JSONB payload so a test can compare what the event recorded
// against what the row actually holds.
func scoreEvents(t *testing.T, conn *pgxpool.Conn, scoreID int32) []struct {
	action string
	notes  string
} {
	t.Helper()
	rows, err := conn.Query(context.Background(), `
		SELECT action, COALESCE(payload->>'notes', '')
		FROM events
		WHERE resource = 'public.scores'
		  AND (payload->>'scoreid')::int = $1
		ORDER BY createdat, eventid
	`, scoreID)
	require.NoError(t, err)
	defer rows.Close()

	var out []struct {
		action string
		notes  string
	}
	for rows.Next() {
		var e struct {
			action string
			notes  string
		}
		require.NoError(t, rows.Scan(&e.action, &e.notes))
		out = append(out, e)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestScoreWriteRecordsEventInSameTransactionIntegration pins the contract the
// four events readers depend on, now that the score path hand-writes its own
// event instead of inheriting one from queryRow's recordEvent hook.
//
// The literals are the whole point. lookupScoreAudit, FindScores' lateral,
// scorediff.go and scoreprogress.go all match on resource = 'public.scores',
// and scoreprogress.go additionally filters action IN ('created','updated').
// Every one of them LEFT JOINs, so a wrong literal yields NULL rather than an
// error - attribution would go dark estate-wide with nothing failing.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestScoreWriteRecordsEventInSameTransactionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	conn, err := db.Conn(context.Background())
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "txevent")

	created := "first answer"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &created,
	}).Save(fx.editorCtx)
	require.NoError(t, err)
	require.NotNil(t, saved)

	events := scoreEvents(t, conn, saved.ScoreID)
	require.Len(t, events, 1, "a create must record exactly one event")
	assert.Equal(t, "created", events[0].action,
		"an insert must record action 'created'; 0048's backfill and scoreprogress.go hardcode the verb")

	// The audit stamp is read back from the committed event rather than
	// synthesized, so it resolves only because the event shares the write's
	// transaction.
	assert.NotNil(t, saved.LastEditedAt, "the response must carry the committed event's timestamp")
	assert.NotNil(t, saved.LastEditedBy, "the response must carry the committed event's editor")

	updated := "second answer"
	_, err = (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &updated,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	events = scoreEvents(t, conn, saved.ScoreID)
	require.Len(t, events, 2, "a real change must record a second event")
	assert.Equal(t, "updated", events[1].action, "an update must record action 'updated'")

	// Read-through PUT of an identical body: the no-op guard returns before any
	// write, so no third event. This is the property ztmf-misc#391 relies on to
	// promise "a read-through PUT creates zero revisions".
	sameNotes := updated
	_, err = (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &sameNotes,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	assert.Len(t, scoreEvents(t, conn, saved.ScoreID), 2,
		"a no-op re-save must record no event")
}

// TestScoreWriteRollsBackWhenEventFailsIntegration pins the widest-blast-radius
// behaviour change in this refactor: an events insert that fails now fails the
// whole write, where recordEvent used to log and swallow it.
//
// That trade is the point - a committed answer with no trace of who changed it
// is the failure mode score history exists to eliminate - but it means a broken
// events table now takes the questionnaire down instead of degrading quietly.
// Nothing else pins it, so it is pinned here.
//
// No fault injection needed: events.userid is a real FK to users, so a context
// carrying a well-formed but non-existent user id makes the event insert raise
// 23503 while the score statement itself is perfectly valid. That is exactly
// the shape of the failure being guarded against - the answer write succeeds,
// its audit does not.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestScoreWriteRollsBackWhenEventFailsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "txrollback")

	// Well-formed and absent from users, so the score write is valid and only
	// the event insert violates its FK.
	ghostCtx := UserToContext(ctx, &User{
		UserID: "00000000-0000-4000-8000-000000000000",
		Role:   "OWNER",
	})

	original := "committed answer"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &original,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	// UPDATE path: the answer must not move when its event cannot be recorded.
	doomed := "should never be committed"
	_, err = (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &doomed,
	}).Save(ghostCtx)
	require.Error(t, err, "a failed event insert must surface as an error, not be swallowed")

	var storedNotes string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT notes FROM scores WHERE scoreid = $1`, saved.ScoreID,
	).Scan(&storedNotes))
	assert.Equal(t, original, storedNotes,
		"the score UPDATE must roll back with its event; a committed answer with no audit row is the state this refactor exists to prevent")

	assert.Len(t, scoreEvents(t, conn, saved.ScoreID), 1,
		"only the original create event may survive")

	// INSERT path: same rule, and the row must not exist at all afterwards.
	var before int
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM scores WHERE datacallid = $1`, fx.dataCallID,
	).Scan(&before))

	orphan := "insert that must not land"
	_, err = (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &orphan,
	}).Save(ghostCtx)
	require.Error(t, err, "a create whose event cannot be recorded must fail")

	var after int
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM scores WHERE datacallid = $1`, fx.dataCallID,
	).Scan(&after))
	assert.Equal(t, before, after, "the rolled-back create must leave no score row behind")
}

// TestScoreConcurrentSavesAgreeWithEventLogIntegration is the regression pin for
// the race the pre-transaction implementation documented and tolerated.
//
// The answer row and its event used to be written on two different pooled
// connections: the UPDATE committed, then recordEvent wrote the event
// afterwards, fire-and-forget. Under concurrency the two orders could invert -
// the row ending on writer A's value while the newest event carried writer B's
// - so last_edited_by named someone whose write had been overwritten. Every
// audit reader projects that newest event, so the questionnaire would attribute
// the visible answer to the wrong person.
//
// SELECT ... FOR UPDATE now serialises writers of one answer and the event
// rides the same transaction, which makes the assertion below exact rather than
// probabilistic: the final row must equal the newest event's payload, and no
// write may lose its event.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestScoreConcurrentSavesAgreeWithEventLogIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "txrace")

	seed := "concurrent seed"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &seed,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	// Eight writers, each a genuinely different answer so none is absorbed by
	// the no-op guard. Kept under the pool's maxConns (16) because each Save
	// holds one connection for its transaction.
	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			notes := fmt.Sprintf("concurrent writer %d", i)
			_, errs[i] = (&Score{
				ScoreID:          saved.ScoreID,
				FismaSystemID:    fx.fismaSystemID,
				FunctionOptionID: fx.functionOptionID,
				DataCallID:       fx.dataCallID,
				Notes:            &notes,
			}).Save(fx.editorCtx)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "concurrent writer %d must not surface an error to its caller", i)
	}

	var storedNotes string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT notes FROM scores WHERE scoreid = $1`, saved.ScoreID,
	).Scan(&storedNotes))

	events := scoreEvents(t, conn, saved.ScoreID)
	require.Len(t, events, writers+1,
		"every committed write must carry exactly one event; a missing one is the dropped-event failure mode")

	assert.Equal(t, storedNotes, events[len(events)-1].notes,
		"the newest event must describe the row that actually survived, or last_edited_by names the wrong person")

	at, by := lookupScoreAudit(ctx, saved.ScoreID)
	assert.NotNil(t, at, "the surviving write must be attributable")
	assert.NotNil(t, by, "the surviving write must name an editor")
}

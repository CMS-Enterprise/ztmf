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

// writePolicy is the undo policy a caller who may write the answer sees. The
// read-only counterpart is asserted explicitly where it matters.
var writePolicy = ScoreUndoPolicy{CanWrite: true}

func revisionCount(t *testing.T, conn *pgxpool.Conn, scoreID int32) int {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM score_revisions WHERE scoreid = $1`, scoreID).Scan(&n))
	return n
}

// TestScoreRevisionsRecordedOnWriteIntegration walks the three write paths that
// produce a revision and the two that must not.
//
// The two negatives are the load-bearing half. A read-through PUT creating a
// revision would fill the drawer with entries the ISSO never made, and the
// rollover creating one per carried answer would write roughly 40 answers x
// ~200 systems of machine noise per cycle - each carrying a second copy of a
// narrative justification, in a table a privacy reviewer is being asked to
// approve.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestScoreRevisionsRecordedOnWriteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revwrite")

	created := "first answer"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &created,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	history, err := FindScoreRevisions(fx.editorCtx, saved, writePolicy)
	require.NoError(t, err)
	require.Len(t, history.Revisions, 1)
	assert.Equal(t, revisionKindCreate, history.Revisions[0].Kind)
	assert.Nil(t, history.Revisions[0].Prev,
		"a create has no earlier value; prev_* is NULL as a set")
	assert.Equal(t, created, derefString(history.Revisions[0].New.Notes))
	assert.Equal(t, scoreStatusDone, history.Revisions[0].New.Status)

	// The create is the head but is not undoable - undo moves between values
	// that existed, and there is no path back to unanswered.
	require.NotNil(t, history.Head)
	assert.False(t, history.Head.Undoable)
	require.NotNil(t, history.Head.Reason)
	assert.Equal(t, reasonIsCreate, *history.Head.Reason)

	updated := "second answer"
	_, err = (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &updated,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	history, err = FindScoreRevisions(fx.editorCtx, saved, writePolicy)
	require.NoError(t, err)
	require.Len(t, history.Revisions, 2)
	assert.Equal(t, revisionKindUpdate, history.Revisions[0].Kind, "newest first")
	assert.Equal(t, created, derefString(history.Revisions[0].Prev.Notes),
		"the update's prev must be the value it replaced")
	assert.Equal(t, updated, derefString(history.Revisions[0].New.Notes))
	assert.True(t, history.Head.Undoable)
	assert.Nil(t, history.Head.Reason)

	// Older rows are permanently non-undoable: undo walks back one step at a
	// time, which is what keeps expected_head_revisionid unambiguous.
	require.NotNil(t, history.Revisions[1].Reason)
	assert.Equal(t, reasonNotHead, *history.Revisions[1].Reason)

	// Negative 1: read-through PUT of an identical body.
	same := updated
	_, err = (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &same,
	}).Save(fx.editorCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, revisionCount(t, conn, saved.ScoreID),
		"a read-through PUT must create zero revisions")

	// Negative 2: the rollover. copyPreviousScores writes through a raw
	// conn.Exec that never enters the revision path.
	var nextDC int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), '2100-06-01T00:00:00Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%srevrollover_%d", integrationTestPrefix, fx.dataCallID)).Scan(&nextDC))

	_, err = copyPreviousScores(fx.editorCtx, nextDC)
	require.NoError(t, err)

	var carriedRevisions int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM score_revisions r
		INNER JOIN scores s ON s.scoreid = r.scoreid
		WHERE s.datacallid = $1
	`, nextDC).Scan(&carriedRevisions))
	assert.Zero(t, carriedRevisions, "the rollover must create zero revisions")
}

// TestScoreUndoRestoresPreviousAnswerIntegration is the core undo contract:
// answer, change, undo, original value is back - and the undo is itself
// recorded rather than popping the head, so who undid what stays auditable and
// redo is just undoing the undo.
func TestScoreUndoRestoresPreviousAnswerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revundo")

	original := "original answer"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &original,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	changed := "changed answer"
	afterChange, err := (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &changed,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	history, err := FindScoreRevisions(fx.editorCtx, afterChange, writePolicy)
	require.NoError(t, err)
	head := history.Head.RevisionID

	result, err := UndoScoreRevision(fx.editorCtx, afterChange, &head)
	require.NoError(t, err)

	assert.Equal(t, original, derefString(result.Score.Notes),
		"undo must restore the value the head revision replaced")

	// Appended, not popped.
	assert.Equal(t, 3, revisionCount(t, conn, saved.ScoreID))
	require.NotNil(t, result.Revision)
	assert.Equal(t, revisionKindUndo, result.Revision.Kind)
	require.NotNil(t, result.Revision.UndoesRevisionID)
	assert.Equal(t, head, *result.Revision.UndoesRevisionID,
		"the undo revision must name what it reverted")

	// The undo is itself the head and is undoable - that is redo, with no
	// separate endpoint.
	require.NotNil(t, result.Head)
	assert.True(t, result.Head.Undoable)
	assert.Equal(t, revisionKindUndo, result.Head.Kind)

	redoHead := result.Head.RevisionID
	redone, err := UndoScoreRevision(fx.editorCtx, result.Score, &redoHead)
	require.NoError(t, err)
	assert.Equal(t, changed, derefString(redone.Score.Notes),
		"undoing the undo must return the value undo removed")

	// The answer is still readable from the row itself, not just the response.
	var storedNotes string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT notes FROM scores WHERE scoreid = $1`, saved.ScoreID).Scan(&storedNotes))
	assert.Equal(t, changed, storedNotes)
}

// TestScoreUndoOfCarriedForwardEditRestoresNotStartedIntegration pins the
// acceptance criterion with the widest visible consequence: undoing the first
// edit of a carried-forward answer returns status to not_started, so the
// progress fraction goes DOWN.
//
// Nothing in the product has ever decremented it before. scoreprogress.go reads
// scores.status so it tolerates the move, but it will look like a bug to anyone
// watching a dashboard, which is why it is asserted rather than assumed.
//
// NOTE: "decrements questionsupdated by exactly one" is only guaranteed once
// ztmf#491's unique index exists. Without it a second 'done' row for the same
// function keeps the count up, and duplicates are live in prod (124 pairs
// across three cycles per ztmf-misc#336). This test asserts the status
// transition, which holds either way, and checks the count against the same
// score's own function rather than the whole system.
func TestScoreUndoOfCarriedForwardEditRestoresNotStartedIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revcarried")

	// A carried-forward row: seeded directly with status not_started, exactly
	// what copyPreviousScores produces.
	carried := "carried forward answer"
	var scoreID int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes, status)
		VALUES ($1, $2, $3, $4, 'not_started')
		RETURNING scoreid
	`, fx.fismaSystemID, fx.functionOptionID, fx.dataCallID, carried).Scan(&scoreID))

	stored, err := FindScoreByID(ctx, scoreID)
	require.NoError(t, err)
	require.Equal(t, scoreStatusNotStarted, stored.Status)

	edited := "edited this cycle"
	afterEdit, err := (&Score{
		ScoreID:          scoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &edited,
	}).Save(fx.editorCtx)
	require.NoError(t, err)
	require.Equal(t, scoreStatusDone, afterEdit.Status, "a real edit marks the answer done")

	history, err := FindScoreRevisions(fx.editorCtx, afterEdit, writePolicy)
	require.NoError(t, err)
	require.Len(t, history.Revisions, 1, "only the edit is recorded; the carried row itself is not a revision")
	require.Equal(t, revisionKindUpdate, history.Revisions[0].Kind)
	assert.Equal(t, scoreStatusNotStarted, history.Revisions[0].Prev.Status,
		"the revision must remember the carried row was not_started")

	head := history.Head.RevisionID
	result, err := UndoScoreRevision(fx.editorCtx, afterEdit, &head)
	require.NoError(t, err)

	assert.Equal(t, scoreStatusNotStarted, result.Score.Status,
		"undoing the first edit of a carried answer must return it to not_started")
	assert.Equal(t, carried, derefString(result.Score.Notes))

	var storedStatus string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT status FROM scores WHERE scoreid = $1`, scoreID).Scan(&storedStatus))
	assert.Equal(t, scoreStatusNotStarted, storedStatus)
}

// TestScoreUndoOfConfirmUnconfirmsIntegration pins the answer to #391's open
// question: undo acts on the head whatever its kind, so undoing a confirm
// un-confirms and changes no answer field.
//
// The alternative - skipping confirm revisions - was rejected because it makes
// the token ambiguous (expected_head_revisionid would name one revision while
// the undo targeted another) and because it surprises: a user who confirms,
// clicks Undo, and sees their ANSWER change has been given the wrong verb.
func TestScoreUndoOfConfirmUnconfirmsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revconfirm")

	carried := "carried answer awaiting confirmation"
	var scoreID int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes, status)
		VALUES ($1, $2, $3, $4, 'not_started')
		RETURNING scoreid
	`, fx.fismaSystemID, fx.functionOptionID, fx.dataCallID, carried).Scan(&scoreID))

	stored, err := FindScoreByID(ctx, scoreID)
	require.NoError(t, err)

	confirmed, err := stored.Confirm(fx.editorCtx)
	require.NoError(t, err)
	require.Equal(t, scoreStatusDone, confirmed.Status)

	history, err := FindScoreRevisions(fx.editorCtx, confirmed, writePolicy)
	require.NoError(t, err)
	require.Len(t, history.Revisions, 1)
	assert.Equal(t, revisionKindConfirm, history.Revisions[0].Kind,
		"a confirm is recorded as its own kind, not as an update")
	assert.Equal(t, scoreStatusNotStarted, history.Revisions[0].Prev.Status)
	assert.Equal(t, scoreStatusDone, history.Revisions[0].New.Status)
	assert.Equal(t, carried, derefString(history.Revisions[0].New.Notes),
		"a confirm changes no answer field")

	head := history.Head.RevisionID
	result, err := UndoScoreRevision(fx.editorCtx, confirmed, &head)
	require.NoError(t, err)

	assert.Equal(t, scoreStatusNotStarted, result.Score.Status,
		"undoing a confirm must un-confirm")
	assert.Equal(t, carried, derefString(result.Score.Notes),
		"undoing a confirm must not change the answer")
	assert.Equal(t, fx.functionOptionID, result.Score.FunctionOptionID)
}

// TestScoreUndoConflictAndGuardsIntegration covers every way an undo is
// refused: a stale token, a missing token, and a create head.
func TestScoreUndoConflictAndGuardsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revconflict")

	first := "first"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &first,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	history, err := FindScoreRevisions(fx.editorCtx, saved, writePolicy)
	require.NoError(t, err)
	createHead := history.Head.RevisionID

	// A create head is not undoable, and says so with the same string the read
	// surfaces.
	_, err = UndoScoreRevision(fx.editorCtx, saved, &createHead)
	var invalid *InvalidInputError
	require.ErrorAs(t, err, &invalid, "a create head must be a field-level 400, not a conflict")
	assert.Equal(t, reasonIsCreate, invalid.Data()["expected_head_revisionid"])

	// A missing token is a field-level 400 too, never a silent last-write-wins.
	_, err = UndoScoreRevision(fx.editorCtx, saved, nil)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "required", invalid.Data()["expected_head_revisionid"])

	second := "second"
	afterChange, err := (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &second,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	// A token naming a revision that is no longer the head: someone else moved
	// the answer in between.
	_, err = UndoScoreRevision(fx.editorCtx, afterChange, &createHead)
	require.ErrorIs(t, err, ErrRevisionConflict)

	var unchanged string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT notes FROM scores WHERE scoreid = $1`, saved.ScoreID).Scan(&unchanged))
	assert.Equal(t, second, unchanged, "a refused undo must change nothing")
	assert.Equal(t, 2, revisionCount(t, conn, saved.ScoreID),
		"a refused undo must append no revision")
}

// TestScoreRevisionsReadOnlyReaderIntegration pins that undoability is
// reader-relative. History stays viewable for audit; only the button goes away.
func TestScoreRevisionsReadOnlyReaderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revreadonly")

	first := "first"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &first,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	second := "second"
	afterChange, err := (&Score{
		ScoreID:          saved.ScoreID,
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &second,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	readOnly, err := FindScoreRevisions(fx.editorCtx, afterChange, ScoreUndoPolicy{CanWrite: false})
	require.NoError(t, err)

	assert.Len(t, readOnly.Revisions, 2, "history stays fully readable for audit")
	require.NotNil(t, readOnly.Head)
	assert.False(t, readOnly.Head.Undoable)
	require.NotNil(t, readOnly.Head.Reason)
	assert.Equal(t, reasonCannotWrite, *readOnly.Head.Reason)
}

// TestScoreConcurrentSavesNumberRevisionsIntegration pins the acceptance
// criterion that two concurrent Saves land revisions 1 and 2 with no
// unique-constraint error reaching either caller.
//
// UNIQUE (scoreid, revision_no) is the backstop, not the mechanism: revision_no
// is MAX+1 read while the writer holds the score row's FOR UPDATE lock, so the
// allocations are serialised rather than racing and retrying.
func TestScoreConcurrentSavesNumberRevisionsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "revconcurrent")

	seed := "seed"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &seed,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			notes := fmt.Sprintf("concurrent revision %d", i)
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
		require.NoError(t, err, "writer %d must not surface a unique-constraint error", i)
	}

	// Contiguous 1..N with no gaps and no duplicates.
	rows, err := conn.Query(ctx,
		`SELECT revision_no FROM score_revisions WHERE scoreid = $1 ORDER BY revision_no`, saved.ScoreID)
	require.NoError(t, err)
	defer rows.Close()

	var seen []int32
	for rows.Next() {
		var n int32
		require.NoError(t, rows.Scan(&n))
		seen = append(seen, n)
	}
	require.NoError(t, rows.Err())

	require.Len(t, seen, writers+1, "every write must have recorded exactly one revision")
	for i, n := range seen {
		assert.Equal(t, int32(i+1), n, "revision numbers must be contiguous from 1")
	}
}

// TestScoreRevisionsConcurrentReadsDoNotExhaustPoolIntegration pins a
// discipline rather than a behaviour: a model function that holds a pooled
// connection must not call another that acquires one.
//
// FindScoreRevisions needs to know whether the data call is still open, and
// dataCallOpen reads datacalls through the pool. Resolving that while holding
// its own connection means every concurrent caller holds one and blocks
// waiting for a second, so at maxConns (16) readers the pool is entirely held
// by waiters and nothing can ever complete - the API stops serving every
// endpoint, not just this one. copyPreviousScores avoids the same trap by
// resolving its data call before taking a connection.
//
// 24 callers is comfortably past the pool ceiling. On the broken ordering this
// does not merely run slowly, it never finishes; the timeout is what fails.
func TestScoreRevisionsConcurrentReadsDoNotExhaustPoolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)

	fx := newTxWriteFixture(t, conn, "revpool")
	notes := "pool probe"
	saved, err := (&Score{
		FismaSystemID:    fx.fismaSystemID,
		FunctionOptionID: fx.functionOptionID,
		DataCallID:       fx.dataCallID,
		Notes:            &notes,
	}).Save(fx.editorCtx)
	require.NoError(t, err)

	// Released before the fan-out: holding one here would mask the very
	// starvation this test exists to catch by changing the arithmetic.
	conn.Release()

	const callers = 24
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = FindScoreRevisions(ctx, saved, writePolicy)
		}(i)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		for i, err := range errs {
			require.NoError(t, err, "caller %d", i)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("concurrent history reads did not complete: a nested pool acquisition has starved the connection pool")
	}
}

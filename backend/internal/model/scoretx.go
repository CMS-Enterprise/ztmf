package model

import (
	"context"
	"errors"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// scoresEventResource is the events.resource literal queryRow's recordEvent hook
// derived from Save's squirrel builder ("public.scores", from Insert/Update
// naming the schema-qualified table). Now that the score write path hand-writes
// its own event inside a transaction, the literal lives here instead of being a
// side effect of how the statement was spelled.
//
// Four readers match on this exact string: lookupScoreAudit, FindScores' lateral,
// scorediff.go and scoreprogress.go. A typo stops attribution estate-wide and
// silently - every one of them LEFT JOINs, so they yield NULL rather than error.
const scoresEventResource = "public.scores"

// scoreTx runs fn inside one transaction on a dedicated pooled connection,
// following the RestoreUser idiom in users.go: begin, resolve-and-release in a
// single defer, commit last.
//
// The score write path needs it because the answer row and its audit event must
// land together or not at all. Before this, the UPDATE committed on one pooled
// connection and recordEvent then wrote the event on a second, fire-and-forget
// with its error discarded - so a dropped event left a score whose edit nothing
// could attribute, which is why scores.status (0048) exists rather than
// progress continuing to be inferred from events.
func scoreTx[T any](ctx context.Context, fn func(pgx.Tx) (*T, error)) (*T, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, trapError(err)
	}
	// Resolve the transaction and then close the dedicated connection in a single
	// defer, so the order cannot be broken by another defer added later. Rollback
	// is a no-op once the transaction has committed.
	defer func() {
		tx.Rollback(ctx)
		conn.Release()
	}()

	res, err := fn(tx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, trapError(err)
	}

	return res, nil
}

// lockScore reads one score row FOR UPDATE, serialising every writer of that
// answer for the rest of the transaction.
//
// Save compares against this row rather than one the controller preloaded, so
// the values it decides "unchanged" against are the values the UPDATE is about
// to overwrite. The preloaded comparison could not promise that: a concurrent
// writer could land a real change between the read and the write, and the
// read-through PUT would then skip a write it should have performed.
//
// The cost is real and deliberate. A read-through PUT - the questionnaire's
// hottest path, issued on every Next click - went from a single round trip to
// BEGIN + SELECT FOR UPDATE + COMMIT, holding a row lock throughout. Do NOT
// reclaim it by comparing against a caller-supplied row before opening the
// transaction: that is precisely the race this closes, and it reads as a
// harmless optimization.
func lockScore(ctx context.Context, tx pgx.Tx, scoreID int32) (*Score, error) {
	current := &Score{}
	err := tx.QueryRow(ctx, `
		SELECT scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) AS datecalculated,
		       notes, notes_is_ai_summary, functionoptionid, datacallid, status
		FROM scores WHERE scoreid = $1 FOR UPDATE
	`, scoreID).Scan(
		&current.ScoreID, &current.FismaSystemID, &current.DateCalculated,
		&current.Notes, &current.NotesIsAISummary, &current.FunctionOptionID, &current.DataCallID,
		&current.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoData
	}
	if err != nil {
		return nil, trapError(err)
	}

	return current, nil
}

// writeScore executes a squirrel-built INSERT or UPDATE on scores inside tx and
// collects its RETURNING row. This is queryRow's body minus the recordEvent
// hook: recordEvent has no notion of transactions, it simply is not reached
// because this never calls queryRow. The score path records its event
// explicitly through insertScoreEvent instead.
//
// The RETURNING list and row scanner are deliberately the same as queryRow's,
// so the *Score handed to insertScoreEvent serializes to the identical JSONB
// payload recordEvent produced. Every audit reader keys on
// payload->>'scoreid', and events_score_audit_idx indexes that expression.
func writeScore(ctx context.Context, tx pgx.Tx, sqlb SqlBuilder) (*Score, error) {
	sql, args, err := sqlb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, trapError(err)
	}

	saved, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[Score])
	if err != nil {
		return nil, trapError(err)
	}

	return &saved, nil
}

// insertScoreEvent writes the audit event recordEvent used to write for this
// path. action must stay within ('created','updated'): migration 0048's
// backfill, the seed's status-sync and scoreprogress.go all hardcode that pair
// as the set meaning "a human answered this cycle", so a new verb would make
// the write invisible to progress rather than differently labelled.
//
// Actor-gated exactly as recordEvent is, so a context carrying no user still
// records nothing - the empire seed and several model integration tests write
// scores that way.
//
// Unlike recordEvent this returns its error rather than discarding it. The event
// now shares the write's transaction and Save reads it straight back through
// lookupScoreAudit, so letting it fail quietly would either advertise audit
// fields the next GET cannot confirm, or commit an answer with no trace of who
// changed it.
func insertScoreEvent(ctx context.Context, tx pgx.Tx, action string, saved *Score) error {
	actor := UserFromContext(ctx)
	if actor == nil {
		return nil
	}

	// createdat is written explicitly as clock_timestamp(), NOT left to the
	// column's CURRENT_TIMESTAMP default. CURRENT_TIMESTAMP is transaction START
	// time in Postgres, and this event is now written inside the write's
	// transaction - one that may have sat waiting on lockScore's FOR UPDATE.
	//
	// Concretely: writer A begins, blocks on the lock; writer B begins later,
	// takes the lock, writes and commits; A then proceeds and commits last, so
	// scores holds A's answer - but A's event carries the EARLIER timestamp and
	// B's looks newest. All four audit readers order by createdat DESC, so
	// last_edited_by would name the writer whose answer was overwritten.
	//
	// This ordering used to be incidentally correct: recordEvent ran after the
	// score had committed, in its own short transaction, so start time was
	// effectively insert time. clock_timestamp() is the actual wall clock at
	// statement execution, and because the insert happens while the row lock is
	// still held, event order now matches commit order by construction.
	// Pinned by TestScoreConcurrentSavesAgreeWithEventLogIntegration.
	_, err := tx.Exec(ctx,
		"INSERT INTO events (userid, action, resource, payload, createdat) VALUES ($1, $2, $3, $4, clock_timestamp())",
		actor.UserID, action, scoresEventResource, saved,
	)

	return trapError(err)
}

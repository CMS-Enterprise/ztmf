package model

import (
	"context"
	"errors"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// Revision kinds. Stored data, pinned by score_revisions_kind_check in
// migration 0060 - changing a VALUE here is a data migration, not an edit.
//
// revisionKindTranslate is reserved for ztmf-misc#397's version re-pin and is
// never written today.
const (
	revisionKindCreate    = "create"
	revisionKindUpdate    = "update"
	revisionKindConfirm   = "confirm"
	revisionKindUndo      = "undo"
	revisionKindTranslate = "translate"
)

// maxScoreRevisions caps the history read. One answer on one open cycle
// accumulates one revision per real edit - tens, not thousands - so this is a
// backstop against a pathological row rather than pagination. If it is ever
// reached, the wrapper shape below makes adding limit/offset additive.
const maxScoreRevisions = 200

// Reasons a revision cannot be undone. Server-owned and rendered verbatim by
// the client, so the undo policy lives in exactly one place; ztmf-misc#392
// displays these strings rather than re-deriving the rules.
const (
	reasonNotHead     = "Only the most recent change can be undone."
	reasonIsCreate    = "This is the original answer - there is no earlier value to return to."
	reasonCannotWrite = "You do not have permission to change this answer."
	reasonCallClosed  = "This data call has closed."
	// An undo is deliberately terminal. Because undo appends rather than pops,
	// redo-of-undo-of-redo writes a growing run of revisions that all describe
	// the same two values, which makes the history unreadable - noise in the
	// audit trail this table exists to provide. The prior value stays visible
	// in the history, so it can be re-entered as an ordinary edit.
	reasonIsUndo = "An undo cannot itself be undone. Change the answer directly instead."
)

// RevisionSide is one side of a change. Field-for-field identical to
// ScoreDiffSide on purpose: ztmf-misc#392 extracts ScoreDiffModal's renderSide
// into a shared component, and matching names make that a rename rather than a
// mapping layer.
//
// OptionName and Score are LEFT JOINed from functionoptions and are nil when
// the option no longer exists - deliberate, since score_revisions carries no FK
// to the catalog so that history survives its edits.
type RevisionSide struct {
	FunctionOptionID int32    `json:"functionoptionid"`
	OptionName       *string  `json:"optionname,omitempty"`
	Score            *float64 `json:"score,omitempty"`
	Notes            *string  `json:"notes"`
	NotesIsAISummary bool     `json:"notes_is_ai_summary"`
	Status           string   `json:"status"`
}

// ScoreRevision is one recorded change to one answer.
type ScoreRevision struct {
	RevisionID       int64         `json:"revisionid"`
	ScoreID          int32         `json:"scoreid"`
	RevisionNo       int32         `json:"revision_no"`
	Kind             string        `json:"kind"`
	CreatedAt        *time.Time    `json:"createdat"`
	Actor            *AuditRef     `json:"actor,omitempty"`
	Prev             *RevisionSide `json:"prev"`
	New              *RevisionSide `json:"new"`
	UndoesRevisionID *int64        `json:"undoes_revisionid,omitempty"`

	// Undoable is reader-relative, not a property of the row: a read-only tier
	// sees false with reasonCannotWrite on every revision. That is what lets
	// ztmf-misc#392 show history read-only for audit without the client knowing
	// the role matrix. Reason is non-nil exactly when Undoable is false.
	Undoable bool    `json:"undoable"`
	Reason   *string `json:"reason,omitempty"`
}

// ScoreRevisionHead is the subset the client needs to label its button and hold
// an optimistic-concurrency token, projected separately from the list because
// that is the only thing it reads. Keeping them decoupled means pagination or a
// filtered history view stays additive.
type ScoreRevisionHead struct {
	RevisionID int64   `json:"revisionid"`
	RevisionNo int32   `json:"revision_no"`
	Kind       string  `json:"kind"`
	Undoable   bool    `json:"undoable"`
	Reason     *string `json:"reason,omitempty"`
}

// ScoreHistory is the GET response: an object, never a bare array, so
// ztmf-misc#394's versionid and any future paging can be added without a
// breaking change.
type ScoreHistory struct {
	ScoreID       int32              `json:"scoreid"`
	FismaSystemID int32              `json:"fismasystemid"`
	DataCallID    int32              `json:"datacallid"`
	Head          *ScoreRevisionHead `json:"head"`
	Revisions     []*ScoreRevision   `json:"revisions"`
}

// ScoreUndoResult is the undo response. It carries the score so the client can
// repaint the answer and its chip without a refetch, and the new head so the
// client can render its state - which for an undo means reporting that it
// cannot itself be undone, rather than offering a further action.
type ScoreUndoResult struct {
	Score    *Score             `json:"score"`
	Revision *ScoreRevision     `json:"revision"`
	Head     *ScoreRevisionHead `json:"head"`
}

// ScoreUndoPolicy carries the caller-dependent inputs to the undoable decision.
// Passed in rather than derived here because authorization lives in the
// controller, and the model has no view of roles or OpDiv grants.
type ScoreUndoPolicy struct {
	CanWrite bool
}

// scoreSnapshot is the before-image a revision records. Taken from the row
// under lock, or nil for a create.
type scoreSnapshot struct {
	functionOptionID int32
	notes            *string
	notesIsAISummary bool
	status           string
}

func snapshotOf(s *Score) *scoreSnapshot {
	if s == nil {
		return nil
	}
	return &scoreSnapshot{
		functionOptionID: s.FunctionOptionID,
		notes:            s.Notes,
		notesIsAISummary: derefBool(s.NotesIsAISummary),
		status:           s.Status,
	}
}

// recordScoreWrite appends the audit event and the revision for one score
// write, inside the caller's transaction. Save, Confirm and Undo all go through
// it so the three cannot drift on the event literals or on revision_no
// allocation, and so ztmf-misc#397 has one place to hook.
//
// No user in context means neither an event nor a revision, mirroring
// recordEvent's early return. Not hypothetical: _test_data_empire.sql and
// several model integration tests write scores with no session user, and
// score_revisions.userid is NOT NULL.
func recordScoreWrite(ctx context.Context, tx pgx.Tx, saved *Score, prev *scoreSnapshot, kind, action string, undoes *int64) error {
	if UserFromContext(ctx) == nil {
		return nil
	}

	if err := insertScoreEvent(ctx, tx, action, saved); err != nil {
		return err
	}

	return insertScoreRevision(ctx, tx, saved, prev, kind, undoes)
}

// insertScoreRevision writes one revision row. revision_no is MAX+1 read in the
// same statement, which is safe because every writer holds the score row's FOR
// UPDATE lock first - on a create the row is brand new and invisible to other
// transactions, so it is always 1.
//
// functionid is derived from the option being written rather than read from
// scores.functionid, which does not exist on main. Once ztmf#594 lands this can
// become saved.FunctionID off the RETURNING row.
func insertScoreRevision(ctx context.Context, tx pgx.Tx, saved *Score, prev *scoreSnapshot, kind string, undoes *int64) error {
	actor := UserFromContext(ctx)
	if actor == nil {
		return nil
	}

	var (
		prevOption *int32
		prevNotes  *string
		prevAI     *bool
		prevStatus *string
	)
	if prev != nil {
		prevOption = &prev.functionOptionID
		prevNotes = prev.notes
		prevAI = &prev.notesIsAISummary
		prevStatus = &prev.status
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO public.score_revisions (
			scoreid, revision_no, fismasystemid, datacallid, functionid, kind,
			prev_functionoptionid, prev_notes, prev_notes_is_ai_summary, prev_status,
			new_functionoptionid, new_notes, new_notes_is_ai_summary, new_status,
			undoes_revisionid, userid, createdat)
		SELECT $1,
		       COALESCE((SELECT MAX(revision_no) FROM public.score_revisions WHERE scoreid = $1), 0) + 1,
		       $2, $3,
		       (SELECT fo.functionid FROM public.functionoptions fo WHERE fo.functionoptionid = $11),
		       $4, $5, $6, $7, $8, $11, $9, $10, $12, $13, $14, clock_timestamp()
	`,
		saved.ScoreID, saved.FismaSystemID, saved.DataCallID,
		kind, prevOption, prevNotes, prevAI, prevStatus,
		saved.Notes, derefBool(saved.NotesIsAISummary), saved.FunctionOptionID,
		saved.Status, undoes, actor.UserID,
	)

	return trapError(err)
}

// scoreRevisionColumns is the shared projection. prev_* and new_* are widened
// with their catalog labels through two LEFT JOINs, which stay LEFT so a
// revision naming a deleted option still reads.
const scoreRevisionColumns = `
	r.revisionid, r.scoreid, r.revision_no, r.kind, r.createdat,
	r.prev_functionoptionid, pfo.optionname, pfo.score,
	r.prev_notes, r.prev_notes_is_ai_summary, r.prev_status,
	r.new_functionoptionid, nfo.optionname, nfo.score,
	r.new_notes, r.new_notes_is_ai_summary, r.new_status,
	r.undoes_revisionid,
	u.userid, u.fullname, u.email, u.role`

const scoreRevisionJoins = `
	FROM public.score_revisions r
	LEFT JOIN public.functionoptions pfo ON pfo.functionoptionid = r.prev_functionoptionid
	LEFT JOIN public.functionoptions nfo ON nfo.functionoptionid = r.new_functionoptionid
	LEFT JOIN public.users u ON u.userid = r.userid`

// scanScoreRevision reads one row of scoreRevisionColumns. Actor is both-or-
// neither on the user id, matching the audit-field rule in FindScores.
func scanScoreRevision(row pgx.Row) (*ScoreRevision, error) {
	var (
		rev        ScoreRevision
		prevOption *int32
		prevName   *string
		prevScore  *float64
		prevNotes  *string
		prevAI     *bool
		prevStatus *string
		newOption  int32
		newName    *string
		newScore   *float64
		newNotes   *string
		newAI      bool
		newStatus  string
		uid        *string
		uname      *string
		uemail     *string
		urole      *string
	)

	if err := row.Scan(
		&rev.RevisionID, &rev.ScoreID, &rev.RevisionNo, &rev.Kind, &rev.CreatedAt,
		&prevOption, &prevName, &prevScore, &prevNotes, &prevAI, &prevStatus,
		&newOption, &newName, &newScore, &newNotes, &newAI, &newStatus,
		&rev.UndoesRevisionID,
		&uid, &uname, &uemail, &urole,
	); err != nil {
		return nil, err
	}

	if prevOption != nil {
		rev.Prev = &RevisionSide{
			FunctionOptionID: *prevOption,
			OptionName:       prevName,
			Score:            prevScore,
			Notes:            prevNotes,
			NotesIsAISummary: derefBool(prevAI),
			Status:           derefString(prevStatus),
		}
	}

	rev.New = &RevisionSide{
		FunctionOptionID: newOption,
		OptionName:       newName,
		Score:            newScore,
		Notes:            newNotes,
		NotesIsAISummary: newAI,
		Status:           newStatus,
	}

	if uid != nil {
		rev.Actor = &AuditRef{
			UserID: *uid,
			Name:   derefString(uname),
			Email:  derefString(uemail),
			Role:   derefString(urole),
		}
	}

	return &rev, nil
}

// applyUndoPolicy stamps Undoable/Reason on a revision. Only the head is ever
// undoable: undo walks back one step at a time, which keeps
// expected_head_revisionid an unambiguous token and keeps the client from
// having to reason about what a mid-history revert would mean.
func applyUndoPolicy(rev *ScoreRevision, isHead bool, policy ScoreUndoPolicy, callOpen bool) {
	reason := func(s string) {
		rev.Undoable = false
		rev.Reason = &s
	}

	// Reader-wide reasons first, then row-specific ones. A caller who cannot
	// write, or whose cycle has closed, gets the same explanation on every row
	// rather than "this is the original answer" on one and "no permission" on
	// the next - the reason is a property of the caller there, not the revision.
	switch {
	case !policy.CanWrite:
		reason(reasonCannotWrite)
	case !callOpen:
		reason(reasonCallClosed)
	case !isHead:
		reason(reasonNotHead)
	case rev.Kind == revisionKindCreate:
		reason(reasonIsCreate)
	case rev.Kind == revisionKindUndo:
		reason(reasonIsUndo)
	default:
		rev.Undoable = true
		rev.Reason = nil
	}
}

// FindScoreRevisions reads one answer's history, newest first.
//
// The receiver must be a row loaded by FindScoreByID: DataCallID decides
// whether the cycle is still open, and the controller authorizes against the
// loaded FismaSystemID rather than a client-asserted one.
func FindScoreRevisions(ctx context.Context, score *Score, policy ScoreUndoPolicy) (*ScoreHistory, error) {
	// Resolved BEFORE a connection is acquired, and deliberately not moved down
	// beside its only use. dataCallOpen reads datacalls through the pool, so
	// calling it while holding a connection means every caller holds one and
	// waits for a second: at maxConns concurrent readers the pool is fully
	// held by waiters and nothing can complete. copyPreviousScores resolves its
	// data call before taking a connection for the same reason.
	callOpen := score.dataCallOpen(ctx)

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}
	defer conn.Release()

	rows, err := conn.Query(ctx,
		`SELECT `+scoreRevisionColumns+scoreRevisionJoins+`
		WHERE r.scoreid = $1
		ORDER BY r.revision_no DESC
		LIMIT $2`,
		score.ScoreID, maxScoreRevisions,
	)
	if err != nil {
		return nil, trapError(err)
	}
	defer rows.Close()

	history := &ScoreHistory{
		ScoreID:       score.ScoreID,
		FismaSystemID: score.FismaSystemID,
		DataCallID:    score.DataCallID,
		Revisions:     []*ScoreRevision{},
	}

	for rows.Next() {
		rev, err := scanScoreRevision(rows)
		if err != nil {
			return nil, trapError(err)
		}
		history.Revisions = append(history.Revisions, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, trapError(err)
	}

	for i, rev := range history.Revisions {
		applyUndoPolicy(rev, i == 0, policy, callOpen)
	}

	if len(history.Revisions) > 0 {
		h := history.Revisions[0]
		history.Head = &ScoreRevisionHead{
			RevisionID: h.RevisionID,
			RevisionNo: h.RevisionNo,
			Kind:       h.Kind,
			Undoable:   h.Undoable,
			Reason:     h.Reason,
		}
	}

	return history, nil
}

// dataCallOpen reports whether this score's cycle still accepts writes for the
// caller. Mirrors validateDeadline's rule, including the admin bypass, so the
// undoable flag cannot promise something the undo would then refuse.
func (s *Score) dataCallOpen(ctx context.Context) bool {
	return s.validateDeadline(ctx) == nil
}

// UndoScoreRevision reverts an answer to the value its head revision replaced,
// APPENDING a kind='undo' revision rather than popping the head. Who undid what
// therefore stays auditable. An undo is NOT itself undoable: see reasonIsUndo
// below - allowing it would let one repeatedly-clicked button write a run of
// revisions all describing the same two values.
//
// expectedHead is the optimistic-concurrency token, compared under the same FOR
// UPDATE lock that serialises Save. A caller whose history is stale takes
// ErrRevisionConflict rather than silently reverting a change it never saw.
//
// Undo restores status alongside the answer, so undoing the first edit of a
// carried-forward row returns it to not_started. That means the progress
// fraction can go DOWN - correct, and tolerated by scoreprogress.go, but it has
// never happened before this.
//
// A kind='create' head is not undoable: undo moves between values that existed
// and there is no path back to unanswered.
//
// The receiver must be a row loaded by FindScoreByID, not a client body:
// DataCallID drives the deadline check and the controller authorized against
// the loaded FismaSystemID. Same contract as Confirm.
// expectedHead is a pointer so a missing field is distinguishable from a zero
// one, and it is required rather than optional: an undo with no token is
// last-write-wins on a destructive action, and the client always holds the head
// from the history it just rendered.
func UndoScoreRevision(ctx context.Context, score *Score, expectedHead *int64) (*ScoreUndoResult, error) {
	if expectedHead == nil {
		return nil, &InvalidInputError{
			data: map[string]any{"expected_head_revisionid": "required"},
		}
	}

	if err := score.validateDeadline(ctx); err != nil {
		return nil, err
	}

	reverted, err := scoreTx(ctx, func(tx pgx.Tx) (*Score, error) {
		current, err := lockScore(ctx, tx, score.ScoreID)
		if err != nil {
			return nil, err
		}

		head, err := lockedHeadRevision(ctx, tx, score.ScoreID)
		if err != nil {
			return nil, err
		}
		// Two distinct conditions, deliberately not collapsed into one code.
		//
		// No history is not a conflict: nothing changed underneath the caller,
		// there is simply nothing to undo - a row carried forward by the
		// rollover, or written before this feature shipped, has no revisions.
		// Answering ErrRevisionConflict would be actively harmful, because
		// ztmf-misc#392 treats that code as "refresh the drawer and let the user
		// retry" - a retry that can never succeed however many times it refreshes.
		if head == nil {
			return nil, &InvalidInputError{
				data: map[string]any{"expected_head_revisionid": "this answer has no recorded history to undo"},
			}
		}
		// A stale token IS a conflict: someone else moved the answer in between,
		// and refreshing against the new head is exactly the right recovery.
		if head.RevisionID != *expectedHead {
			return nil, ErrRevisionConflict
		}
		// The two terminal kinds, refused with the same string the read
		// surfaces so the client never has to author this copy. Enforced here
		// as well as in applyUndoPolicy because the read only decides what to
		// OFFER - a request can name any revision it likes.
		if head.Kind == revisionKindCreate {
			return nil, &InvalidInputError{
				data: map[string]any{"expected_head_revisionid": reasonIsCreate},
			}
		}
		if head.Kind == revisionKindUndo {
			return nil, &InvalidInputError{
				data: map[string]any{"expected_head_revisionid": reasonIsUndo},
			}
		}

		// Defence in depth behind score_revisions_prev_iff_not_create: a
		// non-create revision always has a prev, so reaching here with nil means
		// the table was written by something other than this package. Refuse
		// rather than panic on a nil dereference.
		target := head.Prev
		if target == nil {
			return nil, ErrRevisionConflict
		}

		sqlb := stmntBuilder.
			Update("public.scores").
			SetMap(map[string]any{
				"functionoptionid":    target.FunctionOptionID,
				"notes":               target.Notes,
				"notes_is_ai_summary": target.NotesIsAISummary,
				"status":              target.Status,
			}).
			// Same binding discipline as Save: pin the write to the row the
			// caller was authorized against, not to the id alone.
			Where("scoreid=? AND fismasystemid=? AND datacallid=?", score.ScoreID, current.FismaSystemID, current.DataCallID).
			Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, notes_is_ai_summary, functionoptionid, datacallid, status")

		updated, err := writeScore(ctx, tx, sqlb)
		if err != nil {
			return nil, err
		}

		// action stays 'updated', not a new 'undone' verb: migration 0048's
		// backfill, the seed status-sync and scoreprogress.go all allowlist
		// ('created','updated'), so a new value would make an undone answer
		// report no last-updated rather than read as differently labelled.
		if err := recordScoreWrite(ctx, tx, updated, snapshotOf(current), revisionKindUndo, eventActionUpdated, &head.RevisionID); err != nil {
			return nil, err
		}

		return updated, nil
	})
	if err != nil {
		return nil, err
	}

	if at, by := lookupScoreAudit(ctx, reverted.ScoreID); at != nil && by != nil {
		reverted.LastEditedAt = at
		reverted.LastEditedBy = by
	}

	// Re-read so the response carries the revision exactly as a subsequent GET
	// will render it, rather than a hand-assembled copy that could drift from
	// the projection.
	history, err := FindScoreRevisions(ctx, reverted, ScoreUndoPolicy{CanWrite: true})
	if err != nil {
		return nil, err
	}

	result := &ScoreUndoResult{Score: reverted, Head: history.Head}
	if len(history.Revisions) > 0 {
		result.Revision = history.Revisions[0]
	}

	return result, nil
}

// lockedHeadRevision reads the newest revision for a score inside the caller's
// transaction. No FOR UPDATE of its own: the caller already holds the score
// row's lock, which is what serialises writers, and score_revisions is
// append-only so the head cannot change underneath that lock.
//
// Returns (nil, nil) when the score has no history, which is not an error - a
// row written before this feature shipped, or by the rollover, has none.
func lockedHeadRevision(ctx context.Context, tx pgx.Tx, scoreID int32) (*ScoreRevision, error) {
	row := tx.QueryRow(ctx,
		`SELECT `+scoreRevisionColumns+scoreRevisionJoins+`
		WHERE r.scoreid = $1
		ORDER BY r.revision_no DESC
		LIMIT 1`,
		scoreID,
	)

	rev, err := scanScoreRevision(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, trapError(err)
	}

	return rev, nil
}

package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// versionTranslationJoinSQL is the one statement of how an answer moves to
// another questionnaire version (ztmf-misc#397): from its option fo, along its
// function's lineage to the target version's function, then to that function's
// option with the same score. The rollover and the re-pin both build on it, so
// what the re-pin's preflight checks is what its write does.
//
// It keys on root_functionid rather than on question lineage and edition:
// 0071's functions_root_version_uniq makes a lineage at most one function per
// version, so no answer can fan out or two land on one function, which the
// edition key cannot promise for v1's duplicate editions. #396's lineage rules
// make the two keys agree on every version published since. A function whose
// options repeat a score (only v1 can hold one) still fans out; callers break
// that tie on tfo.functionoptionid, and both candidates carry the same score.
//
// targetExpr is the target version as an expression in the caller's scope.
func versionTranslationJoinSQL(targetExpr string) string {
	return fmt.Sprintf(`JOIN functions f ON f.functionid = fo.functionid
  JOIN functions tf ON tf.root_functionid = f.root_functionid AND tf.versionid = %s
  JOIN functionoptions tfo ON tfo.functionid = tf.functionid AND tfo.score = fo.score`, targetExpr)
}

// repinMapSQL fills repin_map, built once per re-pin and read by every step
// after it, with each answer in data call $1 that translates into version $2
// and its option there. An answer with no row is one the re-pin drops.
var repinMapSQL = `
	INSERT INTO repin_map (scoreid, to_option, to_function)
	SELECT DISTINCT ON (s.scoreid) s.scoreid, tfo.functionoptionid, tfo.functionid
	  FROM scores s
	  JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
	  ` + versionTranslationJoinSQL("$2") + `
	 WHERE s.datacallid = $1
	 ORDER BY s.scoreid, tfo.functionoptionid`

// repinPreflightSQL counts the answers the re-pin would drop that a person has
// touched this cycle: saved (done), with any recorded history, or on a system
// that has marked the call complete.
var repinPreflightSQL = `
	SELECT COUNT(*) FILTER (WHERE s.status = '` + scoreStatusDone + `'
	                           OR EXISTS (SELECT 1 FROM score_revisions r WHERE r.scoreid = s.scoreid)
	                           OR EXISTS (SELECT 1 FROM datacalls_fismasystems c
	                                       WHERE c.datacallid = s.datacallid AND c.fismasystemid = s.fismasystemid))
	  FROM scores s
	 WHERE s.datacallid = $1 AND NOT EXISTS (SELECT 1 FROM repin_map m WHERE m.scoreid = s.scoreid)`

// repinRevisionsSQL appends a translate revision to every translated answer
// that already has history. Without it the head would name an option the
// answer no longer holds, and undo would write that old-version id back.
// functionid is the target function, so #394's trigger records the target
// version. Notes and status are unchanged, so both sides carry them.
var repinRevisionsSQL = `
	INSERT INTO score_revisions (
		scoreid, revision_no, fismasystemid, datacallid, functionid, kind,
		prev_functionoptionid, prev_notes, prev_notes_is_ai_summary, prev_status,
		new_functionoptionid, new_notes, new_notes_is_ai_summary, new_status,
		userid, createdat)
	SELECT s.scoreid, h.revision_no + 1, s.fismasystemid, s.datacallid, m.to_function, '` + revisionKindTranslate + `',
	       s.functionoptionid, s.notes, s.notes_is_ai_summary, s.status,
	       m.to_option, s.notes, s.notes_is_ai_summary, s.status,
	       $1, clock_timestamp()
	  FROM repin_map m
	  JOIN scores s ON s.scoreid = m.scoreid
	  JOIN LATERAL (SELECT MAX(r.revision_no) AS revision_no FROM score_revisions r WHERE r.scoreid = s.scoreid) h
	    ON h.revision_no IS NOT NULL`

// repinRewriteSQL moves each translated answer to its target option.
// trg_scores_set_functionid re-derives functionid; last_updated_at is left
// alone, since a re-pin is not an in-app edit.
var repinRewriteSQL = `
	UPDATE scores s SET functionoptionid = m.to_option FROM repin_map m WHERE s.scoreid = m.scoreid`

// DataCallRepin is a data call after a re-pin, with what happened to its
// answers.
type DataCallRepin struct {
	DataCall
	Translated int64 `json:"translated"`
	Dropped    int64 `json:"dropped"`
}

// RepinDataCall moves an open data call to another published questionnaire
// version. Answers move by lineage and score. One with no counterpart in the
// target is dropped if nobody has touched it this cycle; if anyone has, the
// change waits for the next cycle (409). A structural remap of given answers has
// no cheap inverse.
func RepinDataCall(ctx context.Context, dataCallID, versionID int32) (*DataCallRepin, error) {
	if !isValidIntID(dataCallID) {
		return nil, ErrNoData
	}
	actor := UserFromContext(ctx)
	if actor == nil {
		return nil, errors.New("a re-pin records translate revisions and needs an actor")
	}
	return scoreTx(ctx, func(tx pgx.Tx) (*DataCallRepin, error) {
		// The data call first: every score write holds it FOR SHARE before its
		// score row (#395's lockScoreDataCall), so this waits out saves in
		// flight and blocks new ones before any score is read. Rewriting
		// scores before this lock would deadlock against them. FOR UPDATE, not
		// NO KEY: it also conflicts with the FK check of any insert referencing
		// the call, such as a system marking it complete, which the preflight
		// counts as touched.
		var res DataCallRepin
		pin := func(sql string, args ...any) error {
			rows, err := tx.Query(ctx, sql, args...)
			if err == nil {
				res.DataCall, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[DataCall])
			}
			return trapError(err)
		}
		if err := pin(`SELECT `+dataCallColumnList+` FROM datacalls WHERE datacallid = $1 FOR UPDATE`, dataCallID); err != nil {
			return nil, err
		}
		from, deadline := res.QuestionnaireVersionID, res.Deadline
		// Before the published check: a call keeps its pin after that version is archived.
		if from == versionID {
			return &res, nil
		}
		if err := lockPinnableVersion(ctx, tx, versionID); err != nil {
			return nil, err
		}
		// Unlike a score write, which an admin may make after the deadline, a
		// re-pin is refused for everyone, so a closed cycle keeps its version. To
		// correct one: extend the deadline, re-pin, restore it. validateDeadline's
		// boundary: the deadline instant itself is still open.
		if time.Now().After(deadline) {
			return nil, ErrPastDeadline
		}

		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE repin_map (scoreid INT PRIMARY KEY, to_option INT, to_function INT) ON COMMIT DROP`); err != nil {
			return nil, trapError(err)
		}
		if _, err := tx.Exec(ctx, repinMapSQL, dataCallID, versionID); err != nil {
			return nil, trapError(err)
		}
		var touched int64
		if err := tx.QueryRow(ctx, repinPreflightSQL, dataCallID).Scan(&touched); err != nil {
			return nil, trapError(err)
		}
		if touched > 0 {
			return nil, fmt.Errorf("%w: %d answers given this cycle have no counterpart in version %d",
				ErrVersionNotTranslatable, touched, versionID)
		}
		if err := pin(`UPDATE datacalls SET questionnaire_versionid = $2 WHERE datacallid = $1 RETURNING `+dataCallColumnList,
			dataCallID, versionID); err != nil {
			return nil, err
		}

		if _, err := tx.Exec(ctx, repinRevisionsSQL, actor.UserID); err != nil {
			return nil, trapError(err)
		}
		tag, err := tx.Exec(ctx, repinRewriteSQL)
		if err != nil {
			return nil, trapError(err)
		}
		res.Translated = tag.RowsAffected()
		// The preflight let these through only because nobody touched them: they
		// are carried-forward rows a rollover into this version would have left out.
		tag, err = tx.Exec(ctx, `
			DELETE FROM scores s
			 WHERE s.datacallid = $1 AND NOT EXISTS (SELECT 1 FROM repin_map m WHERE m.scoreid = s.scoreid)`,
			dataCallID)
		if err != nil {
			return nil, trapError(err)
		}
		res.Dropped = tag.RowsAffected()

		return &res, insertCatalogEvent(ctx, tx, eventActionUpdated, "datacalls", map[string]any{
			"datacallid":     dataCallID,
			"from_versionid": from,
			"to_versionid":   versionID,
			"translated":     res.Translated,
			"dropped":        res.Dropped,
		})
	})
}

// lockPinnableVersion holds a version FOR SHARE and requires it published: a
// draft can be edited or discarded under a pin (#396's draft DELETE relies on
// no call pinning one), and an archived version is retired. Lock order is the
// lifecycle's: an archive takes the row FOR NO KEY UPDATE and so waits for the pin.
func lockPinnableVersion(ctx context.Context, tx pgx.Tx, versionID int32) error {
	if versionID == 0 {
		return &InvalidInputError{data: map[string]any{"questionnaire_versionid": "required"}}
	}
	state, err := lockVersion(ctx, tx, versionID, "FOR SHARE")
	if errors.Is(err, ErrNoData) {
		return &InvalidInputError{data: map[string]any{"questionnaire_versionid": fmt.Sprintf("version %d does not exist", versionID)}}
	}
	if err != nil {
		return err
	}
	if state != versionStatePublished {
		return &InvalidInputError{data: map[string]any{"questionnaire_versionid": fmt.Sprintf("a data call can only be pinned to a published version; version %d is %s", versionID, state)}}
	}
	return nil
}

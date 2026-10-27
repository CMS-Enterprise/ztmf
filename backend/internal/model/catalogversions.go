package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Questionnaire version lifecycle (ztmf-misc#396): fork a draft from the
// current published version, edit it, publish it through a validation gate,
// archive what is no longer current. Content writes outside a draft are refused
// by the content models through catalogWrite and catalogDelete.
//
// Lock order, everywhere: the lifecycle lock (lifecycle operations only), then
// the version row, then content rows. Content writers take the version FOR
// SHARE; publish and discard take it FOR UPDATE.

const (
	versionStateDraft     = "draft"
	versionStatePublished = "published"
	versionStateArchived  = "archived"

	versionsEventResource = "questionnaire_versions"
	maxVersionLabelLen    = 100

	// lifecycleLockKey serializes fork, publish, archive and discard, so a fork
	// never copies a parent that a concurrent publish is about to supersede.
	lifecycleLockKey = 396_307
)

type QuestionnaireVersion struct {
	VersionID       int32      `json:"versionid"`
	Label           string     `json:"label"`
	State           string     `json:"state"`
	ParentVersionID *int32     `json:"parent_versionid" db:"parent_versionid"`
	CreatedAt       time.Time  `json:"createdat"`
	PublishedAt     *time.Time `json:"publishedat"`
}

const versionColumns = `versionid, label, state, parent_versionid, createdat, publishedat`

func FindQuestionnaireVersions(ctx context.Context) ([]*QuestionnaireVersion, error) {
	sqlb := stmntBuilder.Select(versionColumns).From("questionnaire_versions").OrderBy("versionid DESC")
	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[QuestionnaireVersion])
}

func FindQuestionnaireVersionByID(ctx context.Context, versionID int32) (*QuestionnaireVersion, error) {
	if !isValidIntID(versionID) {
		return nil, ErrNoData
	}
	sqlb := stmntBuilder.Select(versionColumns).From("questionnaire_versions").Where("versionid=?", versionID)
	return queryRow(ctx, sqlb, pgx.RowToStructByName[QuestionnaireVersion])
}

// ForkQuestionnaireVersion opens a draft copied from the newest published
// version. The copy happens here rather than at publish so a draft can be
// edited, and so the 432-row copy never runs inside a publish while live data
// calls read the same tables.
//
// Rows are mapped old to new by lineage, never by name (the 0045 defect), and
// carry their source's root, not its id: #394's self-FK only accepts a root
// that is its own root. The legacy question-less band is not copied, since the
// publish gate would reject it. A second draft fails on the one-draft index.
func ForkQuestionnaireVersion(ctx context.Context, label string) (*QuestionnaireVersion, error) {
	label = strings.TrimSpace(label)
	if label == "" || len([]rune(label)) > maxVersionLabelLen {
		return nil, &InvalidInputError{data: map[string]any{"label": fmt.Sprintf("required, at most %d characters", maxVersionLabelLen)}}
	}

	return scoreTx(ctx, func(tx pgx.Tx) (*QuestionnaireVersion, error) {
		if err := lockLifecycle(ctx, tx); err != nil {
			return nil, err
		}
		var parent int32
		err := tx.QueryRow(ctx, `
			SELECT versionid FROM questionnaire_versions
			 WHERE state = 'published' ORDER BY versionid DESC LIMIT 1 FOR SHARE`).Scan(&parent)
		if err != nil {
			return nil, trapError(err)
		}

		draft, err := scanVersion(tx.QueryRow(ctx, `
			INSERT INTO questionnaire_versions (label, state, parent_versionid)
			VALUES ($1, 'draft', $2) RETURNING `+versionColumns, label, parent))
		if errors.Is(err, ErrNotUnique) {
			return nil, &InvalidInputError{data: map[string]any{"state": "a draft is already open; publish or discard it before forking another"}}
		}
		if err != nil {
			return nil, err
		}

		// ORDER BY so new ids follow the parent's order.
		for _, stmt := range []string{
			`INSERT INTO questions (question, notesprompt, pillarid, ordr, versionid, root_questionid)
			 SELECT question, notesprompt, pillarid, ordr, $2, root_questionid
			   FROM questions WHERE versionid = $1 ORDER BY questionid`,
			`INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid, ordr, versionid, root_functionid)
			 SELECT f.function, f.description, f.datacenterenvironment, nq.questionid, f.pillarid, f.ordr, $2, f.root_functionid
			   FROM functions f
			  INNER JOIN questions oq ON oq.questionid = f.questionid
			  INNER JOIN questions nq ON nq.root_questionid = oq.root_questionid AND nq.versionid = $2
			  WHERE f.versionid = $1 ORDER BY f.functionid`,
			`INSERT INTO functionoptions (functionid, score, optionname, description, versionid)
			 SELECT nf.functionid, fo.score, fo.optionname, fo.description, $2
			   FROM functionoptions fo
			  INNER JOIN functions ofn ON ofn.functionid = fo.functionid AND ofn.questionid IS NOT NULL
			  INNER JOIN functions nf ON nf.root_functionid = ofn.root_functionid AND nf.versionid = $2
			  WHERE fo.versionid = $1 ORDER BY fo.functionoptionid`,
		} {
			if _, err := tx.Exec(ctx, stmt, parent, draft.VersionID); err != nil {
				return nil, trapError(err)
			}
		}

		// A short copy is how a version silently loses questions, so the
		// transaction refuses one rather than trusting the joins above.
		var short string
		err = tx.QueryRow(ctx, `
			SELECT concat_ws(', ',
			  CASE WHEN (SELECT COUNT(*) FROM questions WHERE versionid = $1)
			         <> (SELECT COUNT(*) FROM questions WHERE versionid = $2) THEN 'questions' END,
			  CASE WHEN (SELECT COUNT(*) FROM functions WHERE versionid = $1 AND questionid IS NOT NULL)
			         <> (SELECT COUNT(*) FROM functions WHERE versionid = $2) THEN 'functions' END,
			  CASE WHEN (SELECT COUNT(*) FROM functionoptions fo INNER JOIN functions f ON f.functionid = fo.functionid
			              WHERE fo.versionid = $1 AND f.questionid IS NOT NULL)
			         <> (SELECT COUNT(*) FROM functionoptions WHERE versionid = $2) THEN 'functionoptions' END)`,
			parent, draft.VersionID).Scan(&short)
		if err != nil {
			return nil, trapError(err)
		}
		if short != "" {
			return nil, fmt.Errorf("fork of version %d copied short (%s); rolled back", parent, short)
		}
		return draft, insertCatalogEvent(ctx, tx, eventActionCreated, versionsEventResource, draft)
	})
}

// PublishQuestionnaireVersion makes a draft the current version once it passes
// the publish gate. FOR UPDATE on the version row waits for any content write
// in flight (they hold it FOR SHARE) and refuses those that follow. There is
// no unpublish, which is why the gate also refuses an empty catalog.
func PublishQuestionnaireVersion(ctx context.Context, versionID int32) (*QuestionnaireVersion, error) {
	return versionTransition(ctx, versionID, func(tx pgx.Tx, state string) (*QuestionnaireVersion, error) {
		if state != versionStateDraft {
			return nil, &InvalidInputError{data: map[string]any{"state": "only a draft can be published; this version is " + state}}
		}
		if err := checkPublishGate(ctx, tx, versionID); err != nil {
			return nil, err
		}
		return scanVersion(tx.QueryRow(ctx, `
			UPDATE questionnaire_versions SET state = 'published', publishedat = clock_timestamp()
			 WHERE versionid = $1 RETURNING `+versionColumns, versionID))
	})
}

// ArchiveQuestionnaireVersion retires a published version that is no longer
// current. Data calls already pinned to it keep resolving it: reads follow the
// pin, not the state.
func ArchiveQuestionnaireVersion(ctx context.Context, versionID int32) (*QuestionnaireVersion, error) {
	return versionTransition(ctx, versionID, func(tx pgx.Tx, state string) (*QuestionnaireVersion, error) {
		if state != versionStatePublished {
			return nil, &InvalidInputError{data: map[string]any{"state": "only a published version can be archived; this version is " + state}}
		}
		var current int32
		if err := tx.QueryRow(ctx, `SELECT MAX(versionid) FROM questionnaire_versions WHERE state = 'published'`).Scan(&current); err != nil {
			return nil, trapError(err)
		}
		if current == versionID {
			return nil, &InvalidInputError{data: map[string]any{"versionid": "the current published version cannot be archived; publish its successor first"}}
		}
		return scanVersion(tx.QueryRow(ctx, `
			UPDATE questionnaire_versions SET state = 'archived'
			 WHERE versionid = $1 RETURNING `+versionColumns, versionID))
	})
}

// DiscardQuestionnaireVersion deletes a draft and everything in it. Safe for
// the same reason draft content DELETE is, which rests on two guards: no data
// call can pin a draft (#397 must refuse a draft pin at create and re-pin) and
// no score can reference a draft's options (#395's write check). The contents
// are snapshotted into the audit trail before they go.
func DiscardQuestionnaireVersion(ctx context.Context, versionID int32) error {
	_, err := versionTransition(ctx, versionID, func(tx pgx.Tx, state string) (*QuestionnaireVersion, error) {
		if state != versionStateDraft {
			return nil, &InvalidInputError{data: map[string]any{"state": "only a draft can be discarded; this version is " + state}}
		}
		if actor := UserFromContext(ctx); actor != nil {
			_, err := tx.Exec(ctx, `
				INSERT INTO events (userid, action, resource, payload, createdat)
				SELECT $1, $2, 'questions', jsonb_build_object('versionid', $3::int, 'questions',
				       COALESCE(jsonb_agg((`+questionSnapshotSQL("q4.questionid")+`) ORDER BY q4.questionid), '[]')), clock_timestamp()
				  FROM questions q4 WHERE q4.versionid = $3`,
				actor.UserID, eventActionDeleted, versionID)
			if err != nil {
				return nil, trapError(err)
			}
		}
		for _, stmt := range []string{
			`DELETE FROM functions WHERE versionid = $1`, // cascades options
			`DELETE FROM questions WHERE versionid = $1`,
		} {
			if _, err := tx.Exec(ctx, stmt, versionID); err != nil {
				return nil, trapError(err)
			}
		}
		return scanVersion(tx.QueryRow(ctx, `DELETE FROM questionnaire_versions WHERE versionid = $1 RETURNING `+versionColumns, versionID))
	})
	return err
}

// versionTransition is the shared frame of publish, archive and discard: the
// lifecycle lock, the version row FOR UPDATE, the change, and its event in
// the same transaction.
func versionTransition(ctx context.Context, versionID int32, change func(pgx.Tx, string) (*QuestionnaireVersion, error)) (*QuestionnaireVersion, error) {
	if !isValidIntID(versionID) {
		return nil, ErrNoData
	}
	return scoreTx(ctx, func(tx pgx.Tx) (*QuestionnaireVersion, error) {
		if err := lockLifecycle(ctx, tx); err != nil {
			return nil, err
		}
		state, err := lockVersion(ctx, tx, versionID, "FOR UPDATE")
		if err != nil {
			return nil, err
		}
		changed, err := change(tx, state)
		if err != nil {
			return nil, err
		}
		action := eventActionUpdated
		if changed.State == versionStateDraft { // only a discard returns the draft row
			action = eventActionDeleted
		}
		return changed, insertCatalogEvent(ctx, tx, action, versionsEventResource, changed)
	})
}

// lineageRootsSQL lists the function lineages that versions other than
// versionExpr have used for the question lineage of questionExpr and the
// edition envExpr. It is the single statement of the lineage rule, used with
// parameters by the write-time check and correlated by the publish gate: #397
// carries answers by question lineage and edition while #395's diff matches by
// function lineage, so a question and edition must stay on a lineage they
// already had.
func lineageRootsSQL(questionExpr, envExpr, versionExpr string) string {
	return fmt.Sprintf(`
	SELECT DISTINCT g.root_functionid FROM functions g
	 INNER JOIN questions gq ON gq.questionid = g.questionid
	 INNER JOIN questions lq ON lq.root_questionid = gq.root_questionid
	 WHERE lq.questionid = %s AND g.datacenterenvironment = %s AND g.versionid <> %s`, questionExpr, envExpr, versionExpr)
}

// publishRule is one publish-gate check. Each query takes the draft's
// versionid as $1 and returns the ids of the offending rows.
type publishRule struct {
	field, problem, sql string
}

var publishRules = []publishRule{
	{"questions", "has no questions; there is no unpublish, so an empty catalog cannot be published", `
		SELECT $1::int WHERE NOT EXISTS (SELECT 1 FROM questions WHERE versionid = $1)`},
	{"functionoptions", "need exactly four options scored 1-4 with no duplicates", `
		SELECT f.functionid FROM functions f
		  LEFT JOIN functionoptions fo ON fo.functionid = f.functionid
		 WHERE f.versionid = $1
		 GROUP BY f.functionid
		HAVING array_agg(fo.score ORDER BY fo.score) FILTER (WHERE fo.score IS NOT NULL) IS DISTINCT FROM ARRAY[1,2,3,4]
		 ORDER BY f.functionid`},
	{"questionid", "have no question", `
		SELECT functionid FROM functions WHERE versionid = $1 AND questionid IS NULL ORDER BY functionid`},
	{"functions", "have no function", `
		SELECT q.questionid FROM questions q
		 WHERE q.versionid = $1 AND NOT EXISTS (SELECT 1 FROM functions f WHERE f.questionid = q.questionid)
		 ORDER BY q.questionid`},
	{"pillarid", "have a pillar that differs from their question's", `
		SELECT f.functionid FROM functions f INNER JOIN questions q ON q.questionid = f.questionid
		 WHERE f.versionid = $1 AND f.pillarid <> q.pillarid ORDER BY f.functionid`},
	{"datacenterenvironment", "share a question and environment with another function in this version", `
		SELECT f.functionid FROM functions f
		 WHERE f.versionid = $1 AND f.questionid IS NOT NULL
		   AND EXISTS (SELECT 1 FROM functions o WHERE o.versionid = f.versionid AND o.functionid <> f.functionid
		                 AND o.questionid = f.questionid AND o.datacenterenvironment = f.datacenterenvironment)
		 ORDER BY f.functionid`},
	{"root_functionid", "start a new lineage where an earlier version already has one for the same question and environment; restore it instead", `
		SELECT f.functionid FROM functions f
		 WHERE f.versionid = $1 AND f.questionid IS NOT NULL
		   AND EXISTS (` + gateLineageRoots + `)
		   AND f.root_functionid NOT IN (` + gateLineageRoots + `)
		 ORDER BY f.functionid`},
	{"lineage", "carry a lineage an earlier version used for a different question or environment", `
		SELECT f.functionid FROM functions f INNER JOIN questions q ON q.questionid = f.questionid
		 WHERE f.versionid = $1
		   AND EXISTS (SELECT 1 FROM functions g INNER JOIN questions gq ON gq.questionid = g.questionid
		                WHERE g.versionid <> $1 AND g.root_functionid = f.root_functionid
		                  AND (gq.root_questionid <> q.root_questionid OR g.datacenterenvironment <> f.datacenterenvironment))
		 ORDER BY f.functionid`},
}

var gateLineageRoots = lineageRootsSQL("f.questionid", "f.datacenterenvironment", "f.versionid")

// maxReportedIDs keeps a 400 body readable when a rule fails on many rows.
const maxReportedIDs = 20

func checkPublishGate(ctx context.Context, tx pgx.Tx, versionID int32) error {
	failures := map[string]any{}
	for _, rule := range publishRules {
		rows, err := tx.Query(ctx, rule.sql, versionID)
		if err != nil {
			return trapError(err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[int32])
		if err != nil {
			return trapError(err)
		}
		if len(ids) == 0 {
			continue
		}
		shown := make([]string, 0, maxReportedIDs)
		for i, id := range ids {
			if i == maxReportedIDs {
				shown = append(shown, fmt.Sprintf("and %d more", len(ids)-maxReportedIDs))
				break
			}
			shown = append(shown, fmt.Sprint(id))
		}
		failures[rule.field] = fmt.Sprintf("%s %s", strings.Join(shown, ", "), rule.problem)
	}
	if len(failures) > 0 {
		return &InvalidInputError{data: failures}
	}
	return nil
}

func lockLifecycle(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lifecycleLockKey)
	return trapError(err)
}

// lockVersion locks a version row and returns its state. An unknown version
// is ErrNoData, which the controllers report as 404.
func lockVersion(ctx context.Context, tx pgx.Tx, versionID int32, mode string) (string, error) {
	var state string
	err := tx.QueryRow(ctx, `SELECT state FROM questionnaire_versions WHERE versionid = $1 `+mode, versionID).Scan(&state)
	if err != nil {
		return "", trapError(err)
	}
	return state, nil
}

// lockDraftVersion is the content writers' guard: it holds the version row FOR
// SHARE until commit, so a concurrent publish or discard waits for the write
// and every write after it sees the new state. A version that no longer exists
// (discarded meanwhile, or never) is not an open draft either.
func lockDraftVersion(ctx context.Context, tx pgx.Tx, versionID int32) error {
	state, err := lockVersion(ctx, tx, versionID, "FOR SHARE")
	if errors.Is(err, ErrNoData) {
		return ErrVersionNotDraft
	}
	if err != nil {
		return err
	}
	if state != versionStateDraft {
		return ErrVersionNotDraft
	}
	return nil
}

// openDraftVersionID is the version a new question lands in when the request
// names none: the single open draft, if there is one.
func openDraftVersionID(ctx context.Context, tx pgx.Tx) (int32, error) {
	var v int32
	err := tx.QueryRow(ctx, `SELECT versionid FROM questionnaire_versions WHERE state = 'draft'`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrVersionNotDraft
	}
	return v, trapError(err)
}

// catalogWrite runs one content write against a draft version. resolve finds
// the version the row belongs to (versionid never changes, so no row lock is
// taken first); the version is then locked as a draft, so a write aimed at a
// published version is refused before any content check; check validates the
// write; after runs follow-up statements in the same transaction. The event is
// recorded after commit, fire-and-forget, as queryRow does.
func catalogWrite[T any](ctx context.Context,
	resolve func(pgx.Tx) (int32, error),
	check func(tx pgx.Tx, version int32) error,
	build func(version int32) SqlBuilder,
	fn pgx.RowToFunc[T],
	after func(pgx.Tx, *T) error,
) (*T, error) {
	var sqlb SqlBuilder
	res, err := scoreTx(ctx, func(tx pgx.Tx) (*T, error) {
		version, err := resolve(tx)
		if err != nil {
			return nil, err
		}
		if err := lockDraftVersion(ctx, tx, version); err != nil {
			return nil, err
		}
		if check != nil {
			if err := check(tx, version); err != nil {
				return nil, err
			}
		}
		sqlb = build(version)
		sql, args, _ := sqlb.ToSql()
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			return nil, trapError(err)
		}
		row, err := pgx.CollectOneRow(rows, fn)
		if err != nil {
			return nil, trapError(err)
		}
		if after != nil {
			if err := after(tx, &row); err != nil {
				return nil, err
			}
		}
		return &row, nil
	})
	if err != nil {
		return nil, err
	}
	recordEvent(ctx, sqlb, *res)
	return res, nil
}

// catalogDelete deletes content from a draft version. versionSQL reads the
// row's version ($1 = id); lockSQL then locks the row itself, after the
// version, so a delete that lost a race is a 404 and the snapshot is stable.
// The event, written in the same transaction before the delete, carries
// snapshotSQL's jsonb: the row and every child row it takes with it.
func catalogDelete(ctx context.Context, resource, versionSQL, lockSQL string, snapshotSQL func(string) string, id int32, del func(pgx.Tx) error) error {
	if !isValidIntID(id) {
		return ErrNoData
	}
	_, err := scoreTx(ctx, func(tx pgx.Tx) (*struct{}, error) {
		version, err := storedVersion(ctx, tx, versionSQL, id)
		if err != nil {
			return nil, err
		}
		if err := lockDraftVersion(ctx, tx, version); err != nil {
			return nil, err
		}
		if _, err := storedVersion(ctx, tx, lockSQL, id); err != nil {
			return nil, err
		}
		if actor := UserFromContext(ctx); actor != nil {
			_, err := tx.Exec(ctx, `
				INSERT INTO events (userid, action, resource, payload, createdat)
				SELECT $1, $2, $3, (`+snapshotSQL("$4")+`), clock_timestamp()`,
				actor.UserID, eventActionDeleted, resource, id)
			if err != nil {
				return nil, trapError(err)
			}
		}
		return &struct{}{}, del(tx)
	})
	return err
}

// Snapshot builders for the delete and discard events: idExpr is the row's id
// as the calling statement can see it ($4 in catalogDelete, a column in discard).
func optionSnapshotSQL(idExpr string) string {
	return `SELECT to_jsonb(o) FROM functionoptions o WHERE o.functionoptionid = ` + idExpr
}

func functionSnapshotSQL(idExpr string) string {
	return `SELECT to_jsonb(f) || jsonb_build_object('options',
		COALESCE((SELECT jsonb_agg(to_jsonb(o) ORDER BY o.functionoptionid) FROM functionoptions o WHERE o.functionid = f.functionid), '[]'))
		FROM functions f WHERE f.functionid = ` + idExpr
}

func questionSnapshotSQL(idExpr string) string {
	return `SELECT to_jsonb(q) || jsonb_build_object('functions',
		COALESCE((SELECT jsonb_agg((` + functionSnapshotSQL("fq.functionid") + `) ORDER BY fq.functionid)
		            FROM functions fq WHERE fq.questionid = q.questionid), '[]'))
		FROM questions q WHERE q.questionid = ` + idExpr
}

// storedVersion reads a stored row's version; a missing row is 404.
func storedVersion(ctx context.Context, tx pgx.Tx, sql string, id int32) (int32, error) {
	var version int32
	if err := tx.QueryRow(ctx, sql, id).Scan(&version); err != nil {
		return 0, trapError(err)
	}
	return version, nil
}

// checkRestoredRoot validates a create that names a lineage to restore: the
// root must be a lineage root in an earlier version, and not already present in
// this one. Lineage is not otherwise writable.
func checkRestoredRoot(ctx context.Context, tx pgx.Tx, field, table, idColumn string, root, version int32) error {
	var isRoot, present bool
	err := tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS (SELECT 1 FROM %[1]s WHERE %[2]s = $1 AND %[3]s = %[2]s AND versionid <> $2),
		       EXISTS (SELECT 1 FROM %[1]s WHERE %[3]s = $1 AND versionid = $2)`, table, idColumn, field),
		root, version).Scan(&isRoot, &present)
	if err != nil {
		return trapError(err)
	}
	if !isRoot {
		return &InvalidInputError{data: map[string]any{field: fmt.Sprintf("%d is not a lineage root in an earlier version", root)}}
	}
	if present {
		return &InvalidInputError{data: map[string]any{field: fmt.Sprintf("lineage %d is already in this version", root)}}
	}
	return nil
}

// deleteDraftRow deletes rows whose version the caller has already locked as a
// draft. A 23503 means another row still points here; the 400 names the row
// and, from the constraint, why.
func deleteDraftRow(ctx context.Context, tx pgx.Tx, field, sql string, id int32) (int64, error) {
	tag, err := tx.Exec(ctx, sql, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		reason := "is still referenced (" + pgErr.ConstraintName + ") and cannot be deleted"
		switch pgErr.ConstraintName {
		case "questions_root_questionid_fkey", "functions_root_functionid_fkey":
			reason = "is the lineage root of a row in another version and cannot be deleted"
		case "scores_functionoptionid_fkey", "scores_functionoption_function_fkey":
			reason = "is referenced by an answer and cannot be deleted"
		case "functions_questionid_fkey", "functions_question_version_fkey":
			reason = "gained a function while being deleted; retry"
		}
		return 0, &InvalidInputError{data: map[string]any{field: fmt.Sprintf("%d %s", id, reason)}}
	}
	if err != nil {
		return 0, trapError(err)
	}
	return tag.RowsAffected(), nil
}

func scanVersion(row pgx.Row) (*QuestionnaireVersion, error) {
	var v QuestionnaireVersion
	err := row.Scan(&v.VersionID, &v.Label, &v.State, &v.ParentVersionID, &v.CreatedAt, &v.PublishedAt)
	if err != nil {
		return nil, trapError(err)
	}
	return &v, nil
}

// insertCatalogEvent writes an audit event in the caller's transaction, so a
// change can never commit without a record of who made it. Actor-gated, and
// createdat is clock_timestamp() for the reason insertScoreEvent gives.
func insertCatalogEvent(ctx context.Context, tx pgx.Tx, action, resource string, payload any) error {
	actor := UserFromContext(ctx)
	if actor == nil {
		return nil
	}
	_, err := tx.Exec(ctx,
		"INSERT INTO events (userid, action, resource, payload, createdat) VALUES ($1, $2, $3, $4, clock_timestamp())",
		actor.UserID, action, resource, payload)
	return trapError(err)
}

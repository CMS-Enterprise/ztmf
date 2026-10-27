package migrations

import (
	"context"
	"errors"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQuestionnaireVersionsMigrationIntegration pins migration 0068 against the
// migrated, empire-seeded database. 0068 ran before the seed loaded, so the
// seeded rows prove the triggers; RoundTrip re-runs down and up over them to
// prove the backfill. Each subtest rolls back its own transaction.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.
func TestQuestionnaireVersionsMigrationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	begin := func(t *testing.T) pgx.Tx {
		t.Helper()
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		return tx
	}

	t.Run("SeedLandsOnBaseline", func(t *testing.T) {
		tx := begin(t)
		assertBaselineOnly(ctx, t, tx)
	})

	t.Run("RootDefaultsToOwnIDButKeepsAnExplicitRoot", func(t *testing.T) {
		tx := begin(t)
		v2 := insertDraftVersion(ctx, t, tx)
		q1, _, _ := insertCatalogChain(ctx, t, tx, 1, nil)

		var root int32
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT root_questionid FROM questions WHERE questionid = $1`, q1).Scan(&root))
		assert.Equal(t, q1, root, "a new lineage roots at its own id")

		q2, _, _ := insertCatalogChain(ctx, t, tx, v2, &q1)
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT root_questionid FROM questions WHERE questionid = $1`, q2).Scan(&root))
		assert.Equal(t, q1, root, "a forked row keeps the lineage it was given")
	})

	// A fork that copies a forked id instead of its root must fail, as must a
	// root that names nothing. One transaction each: the 23503 aborts it.
	t.Run("RootMustBeALineageRoot", func(t *testing.T) {
		for name, rootOf := range map[string]func(q1, q2 int32) int32{
			"ForkedID":   func(_, q2 int32) int32 { return q2 },
			"MissingRow": func(q1, _ int32) int32 { return q1 + 1_000_000 },
		} {
			t.Run(name, func(t *testing.T) {
				tx := begin(t)
				v2 := insertDraftVersion(ctx, t, tx)
				q1, _, _ := insertCatalogChain(ctx, t, tx, 1, nil)
				q2, _, _ := insertCatalogChain(ctx, t, tx, v2, &q1)
				_, err := tx.Exec(ctx, `
					INSERT INTO questions (question, notesprompt, pillarid, versionid, root_questionid)
					VALUES ('0068 fixture', '', (SELECT MIN(pillarid) FROM pillars), $1, $2)`,
					v2, rootOf(q1, q2))
				assertSQLError(t, err, "23503", "questions_root_questionid_fkey")
			})
		}
	})

	// Writers that do not name versionid (Function.Save, FunctionOption.Save)
	// must still land a row in its parent's version.
	t.Run("NewRowsTakeTheirParentsVersion", func(t *testing.T) {
		tx := begin(t)
		v2 := insertDraftVersion(ctx, t, tx)
		q2, _, _ := insertCatalogChain(ctx, t, tx, v2, nil)

		var functionID, functionVersion, optionVersion int32
		require.NoError(t, tx.QueryRow(ctx, `
			INSERT INTO functions (function, datacenterenvironment, questionid, pillarid)
			SELECT '0068 fixture', 'fixture', questionid, pillarid FROM questions WHERE questionid = $1
			RETURNING functionid, versionid`, q2).Scan(&functionID, &functionVersion))
		require.NoError(t, tx.QueryRow(ctx, `
			INSERT INTO functionoptions (functionid, score, optionname) VALUES ($1, 2, 'fixture')
			RETURNING versionid`, functionID).Scan(&optionVersion))
		assert.Equal(t, v2, functionVersion)
		assert.Equal(t, v2, optionVersion)
	})

	t.Run("MissingParentStillFailsOnTheFK", func(t *testing.T) {
		tx := begin(t)
		_, err := tx.Exec(ctx, `
			INSERT INTO functionoptions (functionid, score, optionname)
			VALUES ((SELECT MAX(functionid) + 1000 FROM functions), 1, 'orphan')`)
		var pgErr *pgconn.PgError
		require.True(t, errors.As(err, &pgErr), "got %v", err)
		assert.Equal(t, "23503", pgErr.Code, "not a 23502 from a NULL derived version")
	})

	t.Run("OneDraftAtATime", func(t *testing.T) {
		tx := begin(t)
		insertDraftVersion(ctx, t, tx)
		_, err := tx.Exec(ctx, `INSERT INTO questionnaire_versions (label) VALUES ('second draft')`)
		assertSQLError(t, err, "23505", "questionnaire_versions_one_draft")
	})

	// One transaction per case: the expected 23503 aborts it.
	t.Run("CrossVersionWritesAreRejected", func(t *testing.T) {
		type chain struct{ v2, f1, o1, q2, f2 int32 }
		cases := []struct {
			name       string
			stmt       string
			args       func(c chain) []any
			constraint string
		}{
			{"FunctionOntoOtherVersionsQuestion",
				`UPDATE functions SET questionid = $2 WHERE functionid = $1`,
				func(c chain) []any { return []any{c.f1, c.q2} },
				"functions_question_version_fkey"},
			{"OptionOntoOtherVersionsFunction",
				`UPDATE functionoptions SET functionid = $2 WHERE functionoptionid = $1`,
				func(c chain) []any { return []any{c.o1, c.f2} },
				"functionoptions_function_version_fkey"},
			// questionid is cleared so the question FK cannot be what fires.
			{"FunctionMovedOutFromUnderItsOptions",
				`UPDATE functions SET versionid = $2, questionid = NULL WHERE functionid = $1`,
				func(c chain) []any { return []any{c.f1, c.v2} },
				"functionoptions_function_version_fkey"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				tx := begin(t)
				var c chain
				c.v2 = insertDraftVersion(ctx, t, tx)
				_, c.f1, c.o1 = insertCatalogChain(ctx, t, tx, 1, nil)
				c.q2, c.f2, _ = insertCatalogChain(ctx, t, tx, c.v2, nil)

				_, err := tx.Exec(ctx, tc.stmt, tc.args(c)...)
				assertSQLError(t, err, "23503", tc.constraint)
			})
		}
	})

	t.Run("RevisionTakesItsFunctionsVersionAndRoot", func(t *testing.T) {
		tx := begin(t)
		v2 := insertDraftVersion(ctx, t, tx)
		_, f1, _ := insertCatalogChain(ctx, t, tx, 1, nil)
		_, f2, o2 := insertCatalogChain(ctx, t, tx, v2, nil)
		// f2 forks f1's lineage, so its root differs from its own id.
		_, err := tx.Exec(ctx, `UPDATE functions SET root_functionid = $1 WHERE functionid = $2`, f1, f2)
		require.NoError(t, err)

		// Copies a seeded revision's identity columns; the caller-supplied
		// versionid and root_functionid are deliberately wrong.
		var version, root int32
		err = tx.QueryRow(ctx, `
			INSERT INTO score_revisions (scoreid, revision_no, fismasystemid, datacallid,
			    functionid, kind, prev_functionoptionid, prev_notes, prev_notes_is_ai_summary,
			    prev_status, new_functionoptionid, new_notes, new_notes_is_ai_summary,
			    new_status, userid, versionid, root_functionid)
			SELECT r.scoreid,
			       (SELECT MAX(revision_no) + 1 FROM score_revisions x WHERE x.scoreid = r.scoreid),
			       r.fismasystemid, r.datacallid, $1, 'update',
			       r.new_functionoptionid, r.new_notes, r.new_notes_is_ai_summary, r.new_status,
			       $2, r.new_notes, r.new_notes_is_ai_summary, r.new_status, r.userid, 1, -1
			  FROM score_revisions r ORDER BY r.revisionid LIMIT 1
			RETURNING versionid, root_functionid`, f2, o2).Scan(&version, &root)
		require.NoError(t, err, "the empire seed must carry at least one score revision")
		assert.Equal(t, v2, version)
		assert.Equal(t, f1, root, "the revision records the lineage root, not the function's own id")
	})

	t.Run("AnomalyCheckCatchesAMisPin", func(t *testing.T) {
		tx := begin(t)
		v2 := insertDraftVersion(ctx, t, tx)
		_, err := tx.Exec(ctx, `
			UPDATE datacalls SET questionnaire_versionid = $1
			 WHERE datacallid = (SELECT datacallid FROM scores ORDER BY scoreid LIMIT 1)`, v2)
		require.NoError(t, err)

		got := anomalies(ctx, t, tx)
		assert.Positive(t, got["scores_outside_pinned_version"])
		assert.Equal(t, int64(1), got["datacalls_pinned_to_draft"])
	})

	t.Run("AnomalyCheckCatchesNoPublishedVersion", func(t *testing.T) {
		tx := begin(t)
		_, err := tx.Exec(ctx, `UPDATE questionnaire_versions SET state = 'archived' WHERE versionid = 1`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), anomalies(ctx, t, tx)["no_published_version"])
	})

	t.Run("DownRefusesOnceASecondVersionExists", func(t *testing.T) {
		_, down := migrationSQL(t, "pin the catalog and every data call to a baseline")
		tx := begin(t)
		insertDraftVersion(ctx, t, tx)
		_, err := tx.Exec(ctx, down)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ZTMF394_VERSIONS_EXIST")
	})

	// Same mechanism as 0060's orphan test: replica mode is how dangling
	// catalog rows get in despite the FKs.
	t.Run("UpRefusesOrphanedCatalogReadably", func(t *testing.T) {
		up, down := migrationSQL(t, "pin the catalog and every data call to a baseline")
		tx := begin(t)
		_, err := tx.Exec(ctx, down)
		require.NoError(t, err)
		if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
			t.Skipf("cannot set session_replication_role (needs superuser): %v", err)
		}
		// One dangling row of each kind the guard counts.
		_, err = tx.Exec(ctx, `
			INSERT INTO functions (function, datacenterenvironment, questionid, pillarid)
			VALUES ('0068 orphan', 'fixture', (SELECT MAX(questionid) + 1000 FROM questions),
			        (SELECT MIN(pillarid) FROM pillars))`)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `
			INSERT INTO functionoptions (functionid, score, optionname)
			VALUES ((SELECT MAX(functionid) + 1000 FROM functions), 1, '0068 orphan')`)
		require.NoError(t, err)
		tag, err := tx.Exec(ctx, `
			INSERT INTO score_revisions (scoreid, revision_no, fismasystemid, datacallid,
			    functionid, kind, new_functionoptionid, new_notes_is_ai_summary, new_status, userid)
			SELECT r.scoreid, (SELECT MAX(revision_no) + 1 FROM score_revisions x WHERE x.scoreid = r.scoreid),
			       r.fismasystemid, r.datacallid, (SELECT MAX(functionid) + 1000 FROM functions),
			       'create', r.new_functionoptionid, false, r.new_status, r.userid
			  FROM score_revisions r ORDER BY r.revisionid LIMIT 1`)
		require.NoError(t, err)
		require.EqualValues(t, 1, tag.RowsAffected(), "the empire seed must carry a score revision to copy")
		_, err = tx.Exec(ctx, `SET LOCAL session_replication_role = 'origin'`)
		require.NoError(t, err)

		_, err = tx.Exec(ctx, up)
		require.Error(t, err)
		assert.Contains(t, err.Error(),
			"ZTMF394_ORPHANED_CATALOG: 1 functions name a missing question, 1 functionoptions and 1 score_revisions name a missing function")
	})

	t.Run("RoundTripBackfillsSeededRows", func(t *testing.T) {
		up, down := migrationSQL(t, "pin the catalog and every data call to a baseline")
		tx := begin(t)
		_, err := tx.Exec(ctx, down)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, up)
		require.NoError(t, err, "0068 must apply over a populated catalog")
		assertBaselineOnly(ctx, t, tx)
	})
}

func assertBaselineOnly(ctx context.Context, t *testing.T, tx pgx.Tx) {
	t.Helper()
	var versions, offBaseline, revisions int64
	require.NoError(t, tx.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM questionnaire_versions),
		       (SELECT COUNT(*) FROM questions WHERE versionid <> 1 OR root_questionid <> questionid)
		     + (SELECT COUNT(*) FROM functions WHERE versionid <> 1 OR root_functionid <> functionid)
		     + (SELECT COUNT(*) FROM functionoptions WHERE versionid <> 1)
		     + (SELECT COUNT(*) FROM datacalls WHERE questionnaire_versionid <> 1)
		     + (SELECT COUNT(*) FROM score_revisions r JOIN functions f USING (functionid)
		         WHERE r.versionid <> 1 OR r.root_functionid <> f.functionid),
		       (SELECT COUNT(*) FROM score_revisions)`).Scan(&versions, &offBaseline, &revisions))
	assert.Equal(t, int64(1), versions, "only the baseline version exists")
	assert.Zero(t, offBaseline, "every row is on the baseline with its own id as root")
	assert.Positive(t, revisions, "the seed's revisions are what prove the score_revisions columns")
	for check, n := range anomalies(ctx, t, tx) {
		assert.Zero(t, n, check)
	}
}

func anomalies(ctx context.Context, t *testing.T, tx pgx.Tx) map[string]int64 {
	t.Helper()
	rows, err := tx.Query(ctx, `SELECT check_name, n FROM questionnaire_version_anomalies()`)
	require.NoError(t, err)
	got := map[string]int64{}
	for rows.Next() {
		var name string
		var n int64
		require.NoError(t, rows.Scan(&name, &n))
		got[name] = n
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 4)
	return got
}

func insertDraftVersion(ctx context.Context, t *testing.T, tx pgx.Tx) int32 {
	t.Helper()
	var v int32
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO questionnaire_versions (label, state, parent_versionid)
		VALUES ('0068 fixture', 'draft', 1) RETURNING versionid`).Scan(&v))
	return v
}

// insertCatalogChain adds a question, one function and one option in a version.
func insertCatalogChain(ctx context.Context, t *testing.T, tx pgx.Tx, version int32, rootQuestionID *int32) (questionID, functionID, optionID int32) {
	t.Helper()
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO questions (question, notesprompt, pillarid, versionid, root_questionid)
		VALUES ('0068 fixture', '', (SELECT MIN(pillarid) FROM pillars), $1, $2)
		RETURNING questionid`, version, rootQuestionID).Scan(&questionID))
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO functions (function, datacenterenvironment, questionid, pillarid, versionid)
		SELECT '0068 fixture', 'fixture', $1, pillarid, $2 FROM questions WHERE questionid = $1
		RETURNING functionid`, questionID, version).Scan(&functionID))
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO functionoptions (functionid, score, optionname, versionid)
		VALUES ($1, 1, 'fixture', $2) RETURNING functionoptionid`, functionID, version).Scan(&optionID))
	return questionID, functionID, optionID
}

func assertSQLError(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "want %s on %s, got %v", code, constraint, err)
	assert.Equal(t, code, pgErr.Code)
	assert.Equal(t, constraint, pgErr.ConstraintName)
}

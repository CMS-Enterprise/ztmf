package model

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Questionnaire version lifecycle (ztmf-misc#396). Versions here are labelled
// integration_test_ so purgeIntegrationTestVersions can remove them, published
// ones included, before and after each test.

func versionTestSetup(t *testing.T) (context.Context, *pgxpool.Conn) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	purgeIntegrationTestVersions(t)
	t.Cleanup(func() { purgeIntegrationTestVersions(t) })

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	var userID, role string
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT userid, role FROM users ORDER BY (role = 'OWNER') DESC, userid LIMIT 1`).Scan(&userID, &role))
	return UserToContext(ctx, &User{UserID: userID, Role: role}), conn
}

// minimalDraft opens a draft that passes the publish gate: one question with
// one function carrying the four options scored 1-4. The empire seed's own
// catalog does not pass the gate, so a fork of it cannot be published.
func minimalDraft(t *testing.T, ctx context.Context, conn *pgxpool.Conn, environment string) (versionID, questionID, functionID int32) {
	t.Helper()
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO questionnaire_versions (label) VALUES ($1) RETURNING versionid`,
		fmt.Sprintf("%sminimal", integrationTestPrefix)).Scan(&versionID))
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO questions (question, notesprompt, pillarid, versionid)
		VALUES ('minimal', 'prompt', 1, $1) RETURNING questionid`, versionID).Scan(&questionID))
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
		VALUES ('minimal', 'd', $1, $2, 1) RETURNING functionid`, environment, questionID).Scan(&functionID))
	_, err := conn.Exec(ctx, `
		INSERT INTO functionoptions (functionid, score, optionname, description)
		SELECT $1, s, 'opt' || s, 'd' FROM generate_series(1, 4) s`, functionID)
	require.NoError(t, err)
	return versionID, questionID, functionID
}

func requireInvalidOn(t *testing.T, err error, field string, mentions ...int32) {
	t.Helper()
	var invalid *InvalidInputError
	require.ErrorAs(t, err, &invalid)
	msg, ok := invalid.Data()[field]
	require.True(t, ok, "want a 400 naming %q, got %v", field, invalid.Data())
	for _, id := range mentions {
		assert.Contains(t, fmt.Sprint(msg), fmt.Sprint(id), "the 400 names the offending row")
	}
}

func TestQuestionnaireVersionForkIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	// A legacy question-less function in the baseline must not be copied.
	var legacy int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO functions (function, description, datacenterenvironment, pillarid)
		VALUES ('legacy band fixture', 'd', 'AWS', 1) RETURNING functionid`).Scan(&legacy))
	t.Cleanup(func() { deleteFixtureRow(t, `DELETE FROM functions WHERE functionid = $1`, legacy) })
	_, err := conn.Exec(ctx, `
		INSERT INTO functionoptions (functionid, score, optionname, description)
		SELECT $1, s, 'opt', 'd' FROM generate_series(1, 4) s`, legacy)
	require.NoError(t, err)

	draft, err := ForkQuestionnaireVersion(ctx, integrationTestPrefix+"fork")
	require.NoError(t, err)
	assert.Equal(t, versionStateDraft, draft.State)
	require.NotNil(t, draft.ParentVersionID)
	assert.EqualValues(t, 1, *draft.ParentVersionID)

	// Per table: the multiset of (lineage, content, ordr) matches the parent
	// exactly, legacy band excluded.
	for _, tc := range []struct{ name, sql string }{
		{"questions", `SELECT root_questionid, question, notesprompt, pillarid, ordr FROM questions WHERE versionid = $1`},
		{"functions", `SELECT f.root_functionid, q.root_questionid, f.function, f.description, f.datacenterenvironment, f.pillarid, f.ordr
		                 FROM functions f INNER JOIN questions q ON q.questionid = f.questionid WHERE f.versionid = $1`},
		{"functionoptions", `SELECT f.root_functionid, fo.score, fo.optionname, fo.description
		                       FROM functionoptions fo INNER JOIN functions f ON f.functionid = fo.functionid
		                      WHERE fo.versionid = $1 AND f.questionid IS NOT NULL`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, child := multiset(t, ctx, conn, tc.sql, 1), multiset(t, ctx, conn, tc.sql, draft.VersionID)
			require.NotEmpty(t, parent)
			assert.Equal(t, parent, child)
		})
	}

	var copiedLegacy int
	require.NoError(t, conn.QueryRow(ctx, `SELECT COUNT(*) FROM functions WHERE versionid = $1 AND questionid IS NULL`, draft.VersionID).Scan(&copiedLegacy))
	assert.Zero(t, copiedLegacy, "the legacy question-less band is not copied")

	var events int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM events WHERE resource = 'questionnaire_versions' AND action = 'created'
		   AND (payload->>'versionid')::int = $1`, draft.VersionID).Scan(&events))
	assert.Equal(t, 1, events, "the fork is audited")

	_, err = ForkQuestionnaireVersion(ctx, integrationTestPrefix+"second")
	requireInvalidOn(t, err, "state") // the one-draft index, reported as a 400 that says why

	_, err = ForkQuestionnaireVersion(ctx, "  ")
	requireInvalidOn(t, err, "label")
}

func multiset(t *testing.T, ctx context.Context, conn *pgxpool.Conn, sql string, version int32) []string {
	t.Helper()
	rows, err := conn.Query(ctx, sql, version)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		vals, err := rows.Values()
		require.NoError(t, err)
		out = append(out, fmt.Sprint(vals...))
	}
	require.NoError(t, rows.Err())
	sort.Strings(out)
	return out
}

func TestQuestionnaireVersionContentWritesIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	t.Run("BaselineIsReadOnly", func(t *testing.T) {
		_, err := (&Question{QuestionID: 8001, Question: "q", NotesPrompt: "p", PillarID: 1}).Save(ctx)
		assert.ErrorIs(t, err, ErrVersionNotDraft)
		_, err = (&FunctionOption{FunctionOptionID: 1, FunctionID: 7001, Score: 1, OptionName: "Traditional", Description: "d"}).Save(ctx)
		assert.ErrorIs(t, err, ErrVersionNotDraft)
		assert.ErrorIs(t, DeleteFunctionOption(ctx, 1), ErrVersionNotDraft,
			"a published option is never deleted, which keeps undo's prev option resolvable")
		assert.ErrorIs(t, DeleteQuestion(ctx, 8001), ErrVersionNotDraft)
		_, err = (&Question{Question: "q", NotesPrompt: "p", PillarID: 1}).Save(ctx)
		assert.ErrorIs(t, err, ErrVersionNotDraft, "with no draft open, a new question has nowhere to go")

		// Aimed at a published version, the version is the reason given, not the
		// lineage: the draft check runs before any content check.
		baseline, root := int32(1), int32(8001)
		_, err = (&Question{Question: "q", NotesPrompt: "p", PillarID: 1, VersionID: &baseline, RootQuestionID: &root}).Save(ctx)
		assert.ErrorIs(t, err, ErrVersionNotDraft)
	})

	draft, err := ForkQuestionnaireVersion(ctx, integrationTestPrefix+"content")
	require.NoError(t, err)

	t.Run("NewQuestionLandsInTheOpenDraft", func(t *testing.T) {
		q, err := (&Question{Question: "new", NotesPrompt: "p", PillarID: 1}).Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, q.VersionID)
		assert.Equal(t, draft.VersionID, *q.VersionID)
		require.NotNil(t, q.RootQuestionID)
		assert.Equal(t, q.QuestionID, *q.RootQuestionID, "a new question starts its own lineage")

		q.Question = "edited"
		edited, err := q.Save(ctx)
		require.NoError(t, err)
		assert.Equal(t, "edited", edited.Question)

		f, err := (&Function{Function: "new fn", Description: "d", DataCenterEnvironment: "AWS", QuestionID: &q.QuestionID}).Save(ctx)
		require.NoError(t, err, "a new question has no earlier lineage to restore")
		require.NotNil(t, f.VersionID)
		assert.Equal(t, draft.VersionID, *f.VersionID)

		o, err := (&FunctionOption{FunctionID: f.FunctionID, Score: 1, OptionName: "Traditional", Description: "d"}).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, DeleteFunctionOption(ctx, o.FunctionOptionID))
		require.NoError(t, DeleteQuestion(ctx, q.QuestionID), "takes its function with it")
		_, err = FindFunctionByID(ctx, f.FunctionID)
		assert.ErrorIs(t, err, ErrNoData)
	})

	// A forked question already carries its edition's function lineage.
	var forkedQuestion, forkedFunction, functionRoot int32
	var environment string
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT q.questionid, f.functionid, f.root_functionid, f.datacenterenvironment
		  FROM functions f INNER JOIN questions q ON q.questionid = f.questionid
		 WHERE f.versionid = $1 ORDER BY f.functionid LIMIT 1`, draft.VersionID).
		Scan(&forkedQuestion, &forkedFunction, &functionRoot, &environment))

	t.Run("LineageMustBeRestoredNotRestarted", func(t *testing.T) {
		require.NoError(t, DeleteFunction(ctx, forkedFunction))

		_, err := (&Function{Function: "re-add", Description: "d", DataCenterEnvironment: environment, QuestionID: &forkedQuestion}).Save(ctx)
		requireInvalidOn(t, err, "root_functionid", functionRoot)

		wrong := functionRoot + 100000
		_, err = (&Function{Function: "re-add", Description: "d", DataCenterEnvironment: environment, QuestionID: &forkedQuestion, RootFunctionID: &wrong}).Save(ctx)
		requireInvalidOn(t, err, "root_functionid")

		restored, err := (&Function{Function: "re-add", Description: "d", DataCenterEnvironment: environment, QuestionID: &forkedQuestion, RootFunctionID: &functionRoot}).Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, restored.RootFunctionID)
		assert.Equal(t, functionRoot, *restored.RootFunctionID)

		_, err = (&Function{Function: "twice", Description: "d", DataCenterEnvironment: environment, QuestionID: &forkedQuestion, RootFunctionID: &functionRoot}).Save(ctx)
		requireInvalidOn(t, err, "root_functionid", functionRoot)
	})

	t.Run("QuestionLineageRestore", func(t *testing.T) {
		var root int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT root_questionid FROM questions WHERE questionid = $1`, forkedQuestion).Scan(&root))
		_, err := (&Question{Question: "dup", NotesPrompt: "p", PillarID: 1, RootQuestionID: &root}).Save(ctx)
		requireInvalidOn(t, err, "root_questionid", root)

		notARoot := forkedQuestion // a draft id, not a root in an earlier version
		_, err = (&Question{Question: "bad", NotesPrompt: "p", PillarID: 1, RootQuestionID: &notARoot}).Save(ctx)
		requireInvalidOn(t, err, "root_questionid")
	})

	t.Run("DiscardRemovesTheDraft", func(t *testing.T) {
		require.NoError(t, DiscardQuestionnaireVersion(ctx, draft.VersionID))
		_, err := FindQuestionnaireVersionByID(ctx, draft.VersionID)
		assert.ErrorIs(t, err, ErrNoData)
		var left int
		require.NoError(t, conn.QueryRow(ctx, `SELECT COUNT(*) FROM questions WHERE versionid = $1`, draft.VersionID).Scan(&left))
		assert.Zero(t, left)
		var snapshotted int
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT jsonb_array_length(payload->'questions') FROM events
			 WHERE resource = 'questions' AND action = 'deleted' AND payload ? 'questions'
			   AND (payload->>'versionid')::int = $1`, draft.VersionID).Scan(&snapshotted))
		assert.Positive(t, snapshotted, "the discarded contents are in the audit trail")
		requireInvalidOn(t, DiscardQuestionnaireVersion(ctx, 1), "state") // a published version is never discarded
	})
}

func TestQuestionnaireVersionPublishGateIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	// Each rule is broken alone on an otherwise compliant draft, which is then
	// discarded so the next case can open its own.
	cases := []struct {
		name, field string
		breakIt     func(t *testing.T, v, q, f int32) int32 // returns the id the 400 must name
	}{
		{"ExactlyFourOptionsScored1To4", "functionoptions", func(t *testing.T, v, q, f int32) int32 {
			_, err := conn.Exec(ctx, `DELETE FROM functionoptions WHERE functionid = $1 AND score = 4`, f)
			require.NoError(t, err)
			return f
		}},
		{"NoNullQuestion", "questionid", func(t *testing.T, v, q, f int32) int32 {
			var orphan int32
			require.NoError(t, conn.QueryRow(ctx, `
				INSERT INTO functions (function, description, datacenterenvironment, pillarid, versionid)
				VALUES ('orphan', 'd', 'AWS', 1, $1) RETURNING functionid`, v).Scan(&orphan))
			_, err := conn.Exec(ctx, `INSERT INTO functionoptions (functionid, score, optionname, description)
				SELECT $1, s, 'o', 'd' FROM generate_series(1, 4) s`, orphan)
			require.NoError(t, err)
			return orphan
		}},
		{"EveryQuestionHasAFunction", "functions", func(t *testing.T, v, q, f int32) int32 {
			var empty int32
			require.NoError(t, conn.QueryRow(ctx, `
				INSERT INTO questions (question, notesprompt, pillarid, versionid) VALUES ('empty', 'p', 1, $1) RETURNING questionid`, v).Scan(&empty))
			return empty
		}},
		{"FunctionPillarMatchesItsQuestion", "pillarid", func(t *testing.T, v, q, f int32) int32 {
			_, err := conn.Exec(ctx, `UPDATE functions SET pillarid = 2 WHERE functionid = $1`, f)
			require.NoError(t, err)
			return f
		}},
		{"OneFunctionPerQuestionAndEnvironment", "datacenterenvironment", func(t *testing.T, v, q, f int32) int32 {
			var twin int32
			require.NoError(t, conn.QueryRow(ctx, `
				INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
				SELECT 'twin', 'd', datacenterenvironment, questionid, pillarid FROM functions WHERE functionid = $1
				RETURNING functionid`, f).Scan(&twin))
			_, err := conn.Exec(ctx, `INSERT INTO functionoptions (functionid, score, optionname, description)
				SELECT $1, s, 'o', 'd' FROM generate_series(1, 4) s`, twin)
			require.NoError(t, err)
			return twin
		}},
		{"NotEmpty", "questions", func(t *testing.T, v, q, f int32) int32 {
			_, err := conn.Exec(ctx, `DELETE FROM functions WHERE versionid = $1`, v)
			require.NoError(t, err)
			_, err = conn.Exec(ctx, `DELETE FROM questions WHERE versionid = $1`, v)
			require.NoError(t, err)
			return v
		}},
		{"LineageNotMoved", "lineage", func(t *testing.T, v, q, f int32) int32 {
			// The draft function carries a baseline function's lineage under a
			// question lineage that baseline function never belonged to.
			var root int32
			require.NoError(t, conn.QueryRow(ctx, `SELECT root_functionid FROM functions WHERE versionid = 1 AND questionid IS NOT NULL ORDER BY functionid LIMIT 1`).Scan(&root))
			_, err := conn.Exec(ctx, `UPDATE functions SET root_functionid = $2 WHERE functionid = $1`, f, root)
			require.NoError(t, err)
			return f
		}},
		{"NoNewLineageWhereOneExists", "root_functionid", func(t *testing.T, v, q, f int32) int32 {
			// Make the draft question a restored baseline question in the same
			// environment as one of its baseline functions; the draft function
			// keeps its own new lineage, which is what the rule refuses.
			var root int32
			var env string
			require.NoError(t, conn.QueryRow(ctx, `
				SELECT q.root_questionid, f.datacenterenvironment FROM functions f
				 INNER JOIN questions q ON q.questionid = f.questionid
				 WHERE f.versionid = 1 ORDER BY f.functionid LIMIT 1`).Scan(&root, &env))
			_, err := conn.Exec(ctx, `UPDATE questions SET root_questionid = $2 WHERE questionid = $1`, q, root)
			require.NoError(t, err)
			_, err = conn.Exec(ctx, `UPDATE functions SET datacenterenvironment = $2 WHERE functionid = $1`, f, env)
			require.NoError(t, err)
			return f
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, q, f := minimalDraft(t, ctx, conn, "AWS")
			defer func() { require.NoError(t, DiscardQuestionnaireVersion(ctx, v)) }()

			offending := tc.breakIt(t, v, q, f)
			_, err := PublishQuestionnaireVersion(ctx, v)
			requireInvalidOn(t, err, tc.field, offending)
			var invalid *InvalidInputError
			require.ErrorAs(t, err, &invalid)
			assert.Len(t, invalid.Data(), 1, "only the broken rule reports: %v", invalid.Data())

			got, err := FindQuestionnaireVersionByID(ctx, v)
			require.NoError(t, err)
			assert.Equal(t, versionStateDraft, got.State, "a refused publish changes nothing")
		})
	}
}

// create -> edit in draft -> publish -> edit again (403) -> archive.
func TestQuestionnaireVersionLifecycleIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	// A seeded system and its scoring key, so a call pinned to the new version
	// has something to resolve.
	var systemID int32
	var scoringKey string
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fs.fismasystemid, dce.scoring_key FROM fismasystems fs
		 INNER JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		 WHERE fs.decommissioned = FALSE AND dce.scoring_key IS NOT NULL
		 ORDER BY fs.fismasystemid LIMIT 1`).Scan(&systemID, &scoringKey))

	v2, q2, _ := minimalDraft(t, ctx, conn, scoringKey)
	edited, err := (&Question{QuestionID: q2, Question: "edited in draft", NotesPrompt: "p", PillarID: 1}).Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "edited in draft", edited.Question)

	published, err := PublishQuestionnaireVersion(ctx, v2)
	require.NoError(t, err)
	assert.Equal(t, versionStatePublished, published.State)
	assert.NotNil(t, published.PublishedAt)

	_, err = (&Question{QuestionID: q2, Question: "after publish", NotesPrompt: "p", PillarID: 1}).Save(ctx)
	assert.ErrorIs(t, err, ErrVersionNotDraft, "published content is immutable")
	_, err = PublishQuestionnaireVersion(ctx, v2)
	requireInvalidOn(t, err, "state")

	_, err = ArchiveQuestionnaireVersion(ctx, v2)
	requireInvalidOn(t, err, "versionid")

	var pinned int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline, questionnaire_versionid)
		VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz, 1) RETURNING datacallid`,
		integrationTestPrefix+"pinned_baseline").Scan(&pinned))
	before, err := FindQuestionsByFismaSystem(ctx, systemID, FindQuestionsByFismaSystemInput{DataCallID: &pinned})
	require.NoError(t, err)
	require.NotEmpty(t, before)

	archived, err := ArchiveQuestionnaireVersion(ctx, 1)
	require.NoError(t, err, "the baseline is no longer current once v2 is published")
	assert.Equal(t, versionStateArchived, archived.State)
	t.Cleanup(func() {
		deleteFixtureRow(t, `UPDATE questionnaire_versions SET state = 'published' WHERE versionid = $1`, 1)
	})

	after, err := FindQuestionsByFismaSystem(ctx, systemID, FindQuestionsByFismaSystemInput{DataCallID: &pinned})
	require.NoError(t, err)
	assert.Equal(t, before, after, "a call pinned to an archived version keeps resolving it")

	unpinned, err := FindQuestionsByFismaSystem(ctx, systemID, FindQuestionsByFismaSystemInput{})
	require.NoError(t, err)
	require.Len(t, unpinned, 1)
	assert.Equal(t, q2, unpinned[0].QuestionID, "the newest published version is now current")

	_, err = ArchiveQuestionnaireVersion(ctx, 1)
	requireInvalidOn(t, err, "state")
}

// A row a later version descends from cannot be deleted, and says which. Only
// reachable by putting a published ancestor back into draft, which the API
// never does, so this pins the mapping rather than a user path.
func TestQuestionnaireVersionDeleteRootWithDescendantsIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	v, q, _ := minimalDraft(t, ctx, conn, "AWS")
	_, err := conn.Exec(ctx, `UPDATE questionnaire_versions SET state = 'published', publishedat = NOW() WHERE versionid = $1`, v)
	require.NoError(t, err)
	var child int32
	require.NoError(t, conn.QueryRow(ctx, `
		WITH c AS (INSERT INTO questionnaire_versions (label) VALUES ($1) RETURNING versionid)
		INSERT INTO questions (question, notesprompt, pillarid, versionid, root_questionid)
		SELECT 'child', 'p', 1, c.versionid, $2 FROM c RETURNING versionid`, integrationTestPrefix+"child", q).Scan(&child))
	// The child's version must leave draft before the ancestor can return to it.
	_, err = conn.Exec(ctx, `UPDATE questionnaire_versions SET state = 'archived', publishedat = NOW() WHERE versionid = $1`, child)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `UPDATE questionnaire_versions SET state = 'draft', publishedat = NULL WHERE versionid = $1`, v)
	require.NoError(t, err)

	err = DeleteQuestion(ctx, q)
	requireInvalidOn(t, err, "questionid", q)
	assert.True(t, strings.Contains(fmt.Sprint(err.(*InvalidInputError).Data()["questionid"]), "lineage root"))
}

func TestQuestionnaireVersionLineageRulesIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	draft, err := ForkQuestionnaireVersion(ctx, integrationTestPrefix+"lineage")
	require.NoError(t, err)

	var fn Function
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT functionid, questionid, datacenterenvironment FROM functions
		 WHERE versionid = $1 ORDER BY functionid LIMIT 1`, draft.VersionID).Scan(&fn.FunctionID, &fn.QuestionID, &fn.DataCenterEnvironment))
	var otherQuestion int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT questionid FROM questions WHERE versionid = $1 AND questionid <> $2 ORDER BY questionid LIMIT 1`,
		draft.VersionID, *fn.QuestionID).Scan(&otherQuestion))

	t.Run("UpdateCannotMoveALineage", func(t *testing.T) {
		moved := fn
		moved.Function, moved.Description = "moved", "d"
		moved.QuestionID = &otherQuestion
		_, err := moved.Save(ctx)
		requireInvalidOn(t, err, "questionid")

		// A real scoring key, so this reaches the lineage rule, not the key check.
		var otherKey string
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT scoring_key FROM datacenterenvironments
			 WHERE scoring_key IS NOT NULL AND scoring_key <> $1 ORDER BY scoring_key LIMIT 1`, fn.DataCenterEnvironment).Scan(&otherKey))
		moved.QuestionID = fn.QuestionID
		moved.DataCenterEnvironment = otherKey
		_, err = moved.Save(ctx)
		requireInvalidOn(t, err, "datacenterenvironment")
	})

	t.Run("QuestionPillarCarriesToItsFunctions", func(t *testing.T) {
		var text, prompt string
		require.NoError(t, conn.QueryRow(ctx, `SELECT question, notesprompt FROM questions WHERE questionid = $1`, *fn.QuestionID).Scan(&text, &prompt))
		saved, err := (&Question{QuestionID: *fn.QuestionID, Question: text, NotesPrompt: prompt, PillarID: 3}).Save(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, saved.PillarID)
		var stale int
		require.NoError(t, conn.QueryRow(ctx, `SELECT COUNT(*) FROM functions WHERE questionid = $1 AND pillarid <> 3`, *fn.QuestionID).Scan(&stale))
		assert.Zero(t, stale)
	})

	t.Run("DeleteIsAuditedWithItsChildren", func(t *testing.T) {
		require.NoError(t, DeleteQuestion(ctx, *fn.QuestionID))
		var payload map[string]any
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT payload FROM events WHERE resource = 'questions' AND action = 'deleted'
			   AND (payload->>'questionid')::int = $1`, *fn.QuestionID).Scan(&payload))
		functions, ok := payload["functions"].([]any)
		require.True(t, ok, "the event carries the functions the delete took with it: %v", payload)
		require.NotEmpty(t, functions)
		first := functions[0].(map[string]any)
		assert.EqualValues(t, fn.FunctionID, first["functionid"])
		assert.NotNil(t, first["options"])
	})
}

// An earlier version holding two lineages for one question and environment
// (0068 allows it and reports duplicate_editions) must not make every later
// draft unpublishable: keeping either lineage satisfies the rule.
func TestQuestionnaireVersionDuplicateHistoricalLineageIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)

	old, oldQ, f1 := minimalDraft(t, ctx, conn, "AWS")
	var f2 int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
		VALUES ('second lineage', 'd', 'AWS', $1, 1) RETURNING functionid`, oldQ).Scan(&f2))
	_, err := conn.Exec(ctx, `UPDATE questionnaire_versions SET state = 'archived', publishedat = NOW() WHERE versionid = $1`, old)
	require.NoError(t, err)
	_ = f1

	v, q, f := minimalDraft(t, ctx, conn, "AWS")
	_, err = conn.Exec(ctx, `UPDATE questions SET root_questionid = $2 WHERE questionid = $1`, q, oldQ)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `UPDATE functions SET root_functionid = $2 WHERE functionid = $1`, f, f2)
	require.NoError(t, err)

	published, err := PublishQuestionnaireVersion(ctx, v)
	require.NoError(t, err, "keeping one of two historical lineages is allowed")
	assert.Equal(t, versionStatePublished, published.State)

	var events int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM events WHERE resource = 'questionnaire_versions' AND action = 'updated'
		   AND (payload->>'versionid')::int = $1 AND payload->>'state' = 'published'`, v).Scan(&events))
	assert.Equal(t, 1, events, "the publish is audited in the publish's transaction")
}

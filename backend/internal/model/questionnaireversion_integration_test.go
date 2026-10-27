package model

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// purgeIntegrationTestVersions removes every integration_test_ data call, then
// every integration_test_ questionnaire version with its catalog rows. Data
// calls go first because their answers may reference those rows. Run at the
// start of each test and in cleanup, so a run that failed half way cannot
// leave a draft that blocks the next fork.
func purgeIntegrationTestVersions(t *testing.T) {
	t.Helper()
	purgeIntegrationTestRows(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()
	for _, stmt := range []string{
		`DELETE FROM functions WHERE versionid IN (SELECT versionid FROM questionnaire_versions WHERE label LIKE $1)`, // cascades options
		`DELETE FROM questions WHERE versionid IN (SELECT versionid FROM questionnaire_versions WHERE label LIKE $1)`,
		`DELETE FROM questionnaire_versions WHERE label LIKE $1`,
	} {
		_, err := conn.Exec(ctx, stmt, integrationTestPrefix+"%")
		require.NoError(t, err)
	}
}

// forkIdenticalVersion copies the baseline into a new published version the
// way ztmf-misc#396's fork will: lineage-preserving, legacy question-less band
// skipped. Committed, because the read paths go through the pool; the cleanup
// removes it, and any data call pinned to it, before the next test.
func forkIdenticalVersion(t *testing.T, conn *pgxpool.Conn) int32 {
	t.Helper()
	ctx := context.Background()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var v int32
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO questionnaire_versions (label, state, parent_versionid)
		VALUES ($1, 'draft', 1) RETURNING versionid`,
		fmt.Sprintf("%sfork_%d", integrationTestPrefix, time.Now().UnixNano())).Scan(&v))
	for _, stmt := range []string{
		`INSERT INTO questions (question, notesprompt, pillarid, ordr, versionid, root_questionid)
		 SELECT question, notesprompt, pillarid, ordr, $1, root_questionid
		   FROM questions WHERE versionid = 1`,
		`INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid, ordr, versionid, root_functionid)
		 SELECT f.function, f.description, f.datacenterenvironment, nq.questionid, f.pillarid, f.ordr, $1, f.root_functionid
		   FROM functions f
		  INNER JOIN questions oq ON oq.questionid = f.questionid
		  INNER JOIN questions nq ON nq.root_questionid = oq.root_questionid AND nq.versionid = $1
		  WHERE f.versionid = 1`,
		`INSERT INTO functionoptions (functionid, score, optionname, description, versionid)
		 SELECT nf.functionid, fo.score, fo.optionname, fo.description, $1
		   FROM functionoptions fo
		  INNER JOIN functions ofn ON ofn.functionid = fo.functionid
		  INNER JOIN functions nf ON nf.root_functionid = ofn.root_functionid AND nf.versionid = $1
		  WHERE fo.versionid = 1`,
		`UPDATE questionnaire_versions SET state = 'published', publishedat = NOW() WHERE versionid = $1`,
	} {
		_, err := tx.Exec(ctx, stmt, v)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit(ctx))

	t.Cleanup(func() { purgeIntegrationTestVersions(t) })
	return v
}

// pinnedTwin creates a data call pinned to version v holding src's answers
// translated by (lineage, score), which for an identical fork is 1:1. It takes
// src's deadline, so the reduced-pillar rule (anchored on a deadline) treats
// the two calls alike and the version is the only difference.
func pinnedTwin(t *testing.T, conn *pgxpool.Conn, src, v int32) int32 {
	t.Helper()
	ctx := context.Background()
	var dc int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline, questionnaire_versionid)
		SELECT $1, NOW(), deadline, $2 FROM datacalls WHERE datacallid = $3
		RETURNING datacallid`,
		fmt.Sprintf("%stwin_%d_%d", integrationTestPrefix, src, time.Now().UnixNano()), v, src).Scan(&dc))
	tag, err := conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, datecalculated, notes, notes_is_ai_summary,
		                    functionoptionid, datacallid, status, last_updated_at)
		SELECT s.fismasystemid, s.datecalculated, s.notes, s.notes_is_ai_summary,
		       nfo.functionoptionid, $2, s.status, s.last_updated_at
		  FROM scores s
		 INNER JOIN functionoptions ofo ON ofo.functionoptionid = s.functionoptionid
		 INNER JOIN functions ofn ON ofn.functionid = ofo.functionid
		 INNER JOIN functions nf ON nf.root_functionid = ofn.root_functionid AND nf.versionid = $3
		 INNER JOIN functionoptions nfo ON nfo.functionid = nf.functionid AND nfo.score = ofo.score
		 WHERE s.datacallid = $1`, src, dc, v)
	require.NoError(t, err)
	var srcCount int64
	require.NoError(t, conn.QueryRow(ctx, `SELECT COUNT(*) FROM scores WHERE datacallid = $1`, src).Scan(&srcCount))
	require.Positive(t, srcCount, "the source call must have answers, or identical output proves nothing")
	require.Equal(t, srcCount, tag.RowsAffected(), "every answer must translate into the fork")
	return dc
}

// TestQuestionnaireVersionRegressionBarIntegration is ztmf-misc#395's CI bar:
// fork an identical version, publish it, pin a new call to it, and every read
// site must report exactly what the original call reports.
func TestQuestionnaireVersionRegressionBarIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	purgeIntegrationTestVersions(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	yes := true
	aggregate := func(dc int32) []*ScoreAggregate {
		rows, err := FindScoresAggregate(ctx, FindScoresInput{DataCallID: &dc, IncludePillars: &yes})
		require.NoError(t, err)
		for _, r := range rows {
			r.DataCallID = 0
		}
		return rows
	}
	progress := func(dc int32) []*ScoreProgress {
		rows, err := FindScoreProgress(ctx, FindScoreProgressInput{DataCallID: &dc})
		require.NoError(t, err)
		return rows
	}
	// Compared as a multiset: question ids differ across versions, and the
	// empire seed's tied ordr values leave questionid as the sort tiebreaker.
	answers := func(dc int32) []string {
		rows, err := FindAnswers(ctx, FindAnswersInput{DataCallID: dc})
		require.NoError(t, err)
		out := make([]string, len(rows))
		for i, a := range rows {
			out[i] = fmt.Sprint(a.FismaSystemID, a.FismaAcronym, a.DataCenterEnvironment, a.Pillar, a.Question,
				a.Function, a.Description, deref(a.OptionDescription), deref(a.OptionName), deref(a.Score),
				deref(a.Notes), deref(a.NotesIsAISummary), deref(a.TargetMaturityTier), deref(a.TargetMaturityJustification))
		}
		sort.Strings(out)
		return out
	}

	for _, src := range []int32{103, 104} {
		t.Run(fmt.Sprintf("DataCall%d", src), func(t *testing.T) {
			before := aggregate(src)
			require.NotEmpty(t, before)
			// Absolute, not just src-vs-twin: a predicate that fell open would add
			// the other version's rows to both calls alike and still compare equal.
			beforeProgress, beforeAnswers := progress(src), answers(src)

			// The export's answered branch: a system whose environment maps to no
			// catalog keeps every row of its recorded history (#395 AC).
			unmapped := unmappedSystems(t, conn, src)
			var beforeUnmapped []*Answer
			if len(unmapped) > 0 {
				beforeUnmapped, err = FindAnswers(ctx, FindAnswersInput{DataCallID: src, FismaSystemIDs: unmapped})
				require.NoError(t, err)
				require.NotEmpty(t, beforeUnmapped)
			} else {
				require.NotEqual(t, int32(103), src, "the seed must keep an unmapped system with answers on call 103")
			}

			v := forkIdenticalVersion(t, conn)
			twin := pinnedTwin(t, conn, src, v)

			assert.Equal(t, before, aggregate(src), "a new version must not move a call pinned to the old one")
			assert.Equal(t, beforeProgress, progress(src))
			assert.Equal(t, beforeAnswers, answers(src))
			if len(unmapped) > 0 {
				afterUnmapped, err := FindAnswers(ctx, FindAnswersInput{DataCallID: src, FismaSystemIDs: unmapped})
				require.NoError(t, err)
				assert.Equal(t, beforeUnmapped, afterUnmapped, "an unmapped system's export rows are unchanged")
			}

			assert.Equal(t, before, aggregate(twin), "every pillar and system score is identical on the fork")
			assert.Equal(t, progress(src), progress(twin))
			assert.Equal(t, answers(src), answers(twin), "the export walks the pinned catalog")

			diff, err := FindScoreDiff(ctx, FindScoreDiffInput{FromDataCallID: &src, ToDataCallID: &twin})
			require.NoError(t, err)
			assert.Empty(t, diff, "an identical answer on a forked option is not a change")

			// An answer outside the pin - only possible by writing around the API -
			// contributes no count and no last-updated time.
			var strayID int32
			require.NoError(t, conn.QueryRow(ctx, `
				INSERT INTO scores (fismasystemid, functionoptionid, datacallid, status, last_updated_at)
				SELECT s.fismasystemid, ofo.functionoptionid, s.datacallid, 'done', '2099-01-01T00:00:00Z'
				  FROM scores s
				 INNER JOIN functions nf ON nf.functionid = s.functionid
				 INNER JOIN functions ofn ON ofn.root_functionid = nf.root_functionid AND ofn.versionid = 1
				 INNER JOIN functionoptions ofo ON ofo.functionid = ofn.functionid
				 WHERE s.datacallid = $1
				 ORDER BY s.scoreid, ofo.score DESC LIMIT 1
				RETURNING scoreid`, twin).Scan(&strayID))
			assert.Equal(t, progress(src), progress(twin), "counts and lastupdatedat ignore an out-of-version answer")
			assert.Equal(t, before, aggregate(twin))
			_, err = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, strayID)
			require.NoError(t, err)

			// One real change still surfaces, keyed by lineage.
			var changedRoot int32
			require.NoError(t, conn.QueryRow(ctx, `
				UPDATE scores s SET functionoptionid = alt.functionoptionid
				  FROM functionoptions cur, functionoptions alt
				 WHERE s.scoreid = (SELECT MIN(scoreid) FROM scores WHERE datacallid = $1)
				   AND cur.functionoptionid = s.functionoptionid
				   AND alt.functionid = cur.functionid AND alt.score <> cur.score
				   AND alt.functionoptionid = (SELECT MIN(a2.functionoptionid) FROM functionoptions a2
				                                WHERE a2.functionid = cur.functionid AND a2.score <> cur.score)
				RETURNING (SELECT root_functionid FROM functions WHERE functionid = cur.functionid)`,
				twin).Scan(&changedRoot))
			diff, err = FindScoreDiff(ctx, FindScoreDiffInput{FromDataCallID: &src, ToDataCallID: &twin})
			require.NoError(t, err)
			require.Len(t, diff, 1)
			assert.Equal(t, changedRoot, diff[0].RootFunctionID)
			assert.NotEqual(t, diff[0].From.Score, diff[0].To.Score, "a real change is a different score, not just a new id")
			assert.NotNil(t, diff[0].Pillar, "rows carry their pillar for grouping across versions")
		})
	}
}

func unmappedSystems(t *testing.T, conn *pgxpool.Conn, dataCallID int32) []*int32 {
	t.Helper()
	rows, err := conn.Query(context.Background(), `
		SELECT DISTINCT s.fismasystemid FROM scores s
		 INNER JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		  LEFT JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		 WHERE s.datacallid = $1 AND dce.scoring_key IS NULL
		 ORDER BY 1`, dataCallID)
	require.NoError(t, err)
	defer rows.Close()
	var ids []*int32
	for rows.Next() {
		var id int32
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, &id)
	}
	require.NoError(t, rows.Err())
	return ids
}

func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// Reads that name no data call resolve the newest published version, and an
// explicit versionid still reaches an older one.
func TestQuestionnaireVersionUnpinnedReadsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	purgeIntegrationTestVersions(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	v := forkIdenticalVersion(t, conn)

	versionsOf := func(table, idColumn string, ids []int32) map[int32]int {
		got := map[int32]int{}
		rows, err := conn.Query(ctx, fmt.Sprintf(`SELECT versionid FROM %s WHERE %s = ANY($1)`, table, idColumn), ids)
		require.NoError(t, err)
		defer rows.Close()
		for rows.Next() {
			var version int32
			require.NoError(t, rows.Scan(&version))
			got[version]++
		}
		require.NoError(t, rows.Err())
		return got
	}

	fns, err := FindFunctions(ctx, FindFunctionsInput{})
	require.NoError(t, err)
	fnIDs := make([]int32, len(fns))
	for i, f := range fns {
		fnIDs[i] = f.FunctionID
	}
	assert.Equal(t, map[int32]int{v: len(fns)}, versionsOf("functions", "functionid", fnIDs))

	baseline := int32(1)
	oldFns, err := FindFunctions(ctx, FindFunctionsInput{VersionID: &baseline})
	require.NoError(t, err)
	oldIDs := make([]int32, len(oldFns))
	for i, f := range oldFns {
		oldIDs[i] = f.FunctionID
	}
	assert.Equal(t, map[int32]int{1: len(oldFns)}, versionsOf("functions", "functionid", oldIDs), "an explicit versionid still reaches the baseline")

	qs, err := FindQuestions(ctx, FindQuestionsInput{})
	require.NoError(t, err)
	qIDs := make([]int32, len(qs))
	for i, q := range qs {
		qIDs[i] = q.QuestionID
	}
	assert.Equal(t, map[int32]int{v: len(qs)}, versionsOf("questions", "questionid", qIDs))

	var sys, pinnedToBaseline int32
	require.NoError(t, conn.QueryRow(ctx, `SELECT fismasystemid, datacallid FROM scores WHERE datacallid = 103 ORDER BY scoreid LIMIT 1`).Scan(&sys, &pinnedToBaseline))
	unpinned, err := FindQuestionsByFismaSystem(ctx, sys, FindQuestionsByFismaSystemInput{})
	require.NoError(t, err)
	pinned, err := FindQuestionsByFismaSystem(ctx, sys, FindQuestionsByFismaSystemInput{DataCallID: &pinnedToBaseline})
	require.NoError(t, err)
	require.NotEmpty(t, unpinned)
	require.NotEmpty(t, pinned)
	// One version each, never both at once: a doubled questionnaire is the
	// failure a fall-open predicate would produce.
	assert.Equal(t, map[int32]int{v: len(unpinned)}, versionsOf("functions", "functionid", functionIDs(unpinned)))
	assert.Equal(t, map[int32]int{1: len(pinned)}, versionsOf("functions", "functionid", functionIDs(pinned)))

	unknown := int32(-1)
	none, err := FindQuestionsByFismaSystem(ctx, sys, FindQuestionsByFismaSystemInput{DataCallID: &unknown})
	require.NoError(t, err)
	assert.Empty(t, none, "an unresolvable data call fails closed rather than listing every version")
}

func functionIDs(qs []*Question) []int32 {
	ids := make([]int32, len(qs))
	for i, q := range qs {
		ids[i] = q.Function.FunctionID
	}
	return ids
}

// The write path refuses an option outside the data call's pinned version.
func TestQuestionnaireVersionWritePathIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	purgeIntegrationTestVersions(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	v := forkIdenticalVersion(t, conn)

	// Two options of one baseline function, and their v2 counterparts.
	var oldA, oldB, newA, newB int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT oa.functionoptionid, ob.functionoptionid, na.functionoptionid, nb.functionoptionid
		  FROM functions ofn
		 INNER JOIN functions nfn ON nfn.root_functionid = ofn.root_functionid AND nfn.versionid = $1
		 INNER JOIN functionoptions oa ON oa.functionid = ofn.functionid AND oa.score = 1
		 INNER JOIN functionoptions ob ON ob.functionid = ofn.functionid AND ob.score = 2
		 INNER JOIN functionoptions na ON na.functionid = nfn.functionid AND na.score = 1
		 INNER JOIN functionoptions nb ON nb.functionid = nfn.functionid AND nb.score = 2
		 WHERE ofn.versionid = 1
		 ORDER BY ofn.functionid LIMIT 1`, v).Scan(&oldA, &oldB, &newA, &newB))

	assertOutsideVersion := func(t *testing.T, err error) {
		t.Helper()
		var invalid *InvalidInputError
		require.ErrorAs(t, err, &invalid)
		assert.Equal(t, errOptionOutsidePinnedVersion().(*InvalidInputError).Data()["functionoptionid"],
			invalid.Data()["functionoptionid"], "the version is the reason given, not 'different question'")
	}
	answersToRoot := func(t *testing.T, dataCallID, systemID, optionID int32) int {
		t.Helper()
		var n int
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM scores s
			 INNER JOIN functions f ON f.functionid = s.functionid
			 WHERE s.datacallid = $1 AND s.fismasystemid = $2
			   AND f.root_functionid = (SELECT f2.root_functionid FROM functionoptions fo
			                             INNER JOIN functions f2 ON f2.functionid = fo.functionid
			                             WHERE fo.functionoptionid = $3)`,
			dataCallID, systemID, optionID).Scan(&n))
		return n
	}

	// Each case builds its own data call, pinned to the baseline, so any one
	// can run alone.
	t.Run("CreateWithAnotherVersionsOption", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqcreate")
		_, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: newA, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		assertOutsideVersion(t, err)
	})

	// The duplicate this check exists to stop: another version's copy of a
	// question already answered here has its own functionid, so the natural-key
	// lookup misses the existing answer and 0061's index cannot see the clash.
	t.Run("CreateBesideAnExistingAnswer", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqdup")
		_, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: oldA, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		require.NoError(t, err)
		_, err = (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: newB, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		assertOutsideVersion(t, err)
		assert.Equal(t, 1, answersToRoot(t, fx.dataCallID, fx.fismaSystemID, oldA))
	})

	t.Run("CreateWithADraftsOption", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqdraft")
		var draftOption int32
		require.NoError(t, conn.QueryRow(ctx, `
			WITH d AS (INSERT INTO questionnaire_versions (label) VALUES ($1) RETURNING versionid),
			     q AS (INSERT INTO questions (question, notesprompt, pillarid, versionid)
			           SELECT 'draft', '', (SELECT MIN(pillarid) FROM pillars), versionid FROM d RETURNING questionid, pillarid, versionid),
			     f AS (INSERT INTO functions (function, datacenterenvironment, questionid, pillarid, versionid)
			           SELECT 'draft', 'fixture', questionid, pillarid, versionid FROM q RETURNING functionid, versionid)
			INSERT INTO functionoptions (functionid, score, optionname, versionid)
			SELECT functionid, 1, 'draft', versionid FROM f RETURNING functionoptionid`,
			integrationTestPrefix+"draft").Scan(&draftOption))

		_, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: draftOption, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		assertOutsideVersion(t, err)
	})

	// Same question, other version: the version check runs before the
	// different-question check, so this exercises the UPDATE branch's own guard.
	t.Run("UpdateToAnotherVersionsOption", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqupdate")
		saved, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: oldA, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		require.NoError(t, err)
		_, err = (&Score{ScoreID: saved.ScoreID, FismaSystemID: fx.fismaSystemID, FunctionOptionID: newB, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		assertOutsideVersion(t, err)
		assert.Equal(t, oldA, mustFindScore(t, saved.ScoreID).FunctionOptionID, "a refused write changes nothing")
	})

	t.Run("ConfirmAStoredOptionFromAnotherVersion", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqconfirm")
		var scoreID int32
		require.NoError(t, conn.QueryRow(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid, status)
			VALUES ($1, $2, $3, 'not_started') RETURNING scoreid`,
			fx.fismaSystemID, newB, fx.dataCallID).Scan(&scoreID))
		_, err := mustFindScore(t, scoreID).Confirm(fx.editorCtx)
		assertOutsideVersion(t, err)
	})

	t.Run("UndoToAnotherVersionsOption", func(t *testing.T) {
		fx := newTxWriteFixture(t, conn, "vqundo")
		_, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: oldA, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
		require.NoError(t, err)
		twin := pinnedTwin(t, conn, fx.dataCallID, v)
		var scoreID int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT scoreid FROM scores WHERE datacallid = $1 AND functionoptionid = $2`, twin, newA).Scan(&scoreID))
		_, err = (&Score{ScoreID: scoreID, FismaSystemID: fx.fismaSystemID, FunctionOptionID: newB, DataCallID: twin}).Save(fx.editorCtx)
		require.NoError(t, err)
		// A head whose prev names the baseline's option, as a re-pin that skipped
		// its translate revision would leave behind.
		_, err = conn.Exec(ctx, `
			UPDATE score_revisions SET prev_functionoptionid = $2
			 WHERE revisionid = (SELECT MAX(revisionid) FROM score_revisions WHERE scoreid = $1)`, scoreID, oldA)
		require.NoError(t, err)

		stored := mustFindScore(t, scoreID)
		history, err := FindScoreRevisions(fx.editorCtx, stored, writePolicy)
		require.NoError(t, err)
		head := history.Head.RevisionID
		_, err = UndoScoreRevision(fx.editorCtx, stored, &head)
		var invalid *InvalidInputError
		require.ErrorAs(t, err, &invalid)
		assert.Equal(t, reasonOtherVersion, invalid.Data()["expected_head_revisionid"])
		assert.Equal(t, newB, mustFindScore(t, scoreID).FunctionOptionID)
	})
}

// Every scoreid write path must lock the data call before the score row, the
// order the create path takes, so ztmf-misc#397's re-pin (data call row first,
// then scores) waits for them instead of deadlocking. A stand-in re-pin holds
// the data call FOR UPDATE; each path must then block before the score row,
// which a third transaction proves by locking that row NOWAIT.
func TestScoreWritesLockTheDataCallFirstIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	purgeIntegrationTestVersions(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fx := newTxWriteFixture(t, conn, "vqlockorder")
	var other int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT o.functionoptionid FROM functionoptions o
		 INNER JOIN functionoptions fo ON fo.functionid = o.functionid
		 WHERE fo.functionoptionid = $1 AND o.functionoptionid <> $1
		 ORDER BY o.functionoptionid LIMIT 1`, fx.functionOptionID).Scan(&other))

	saved, err := (&Score{FismaSystemID: fx.fismaSystemID, FunctionOptionID: fx.functionOptionID, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
	require.NoError(t, err)

	paths := []struct {
		name  string
		setup func(t *testing.T)
		run   func() error
	}{
		{"UpdateWithAChangedOption", nil, func() error {
			_, err := (&Score{ScoreID: saved.ScoreID, FismaSystemID: fx.fismaSystemID, FunctionOptionID: other, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
			return err
		}},
		{"Confirm", func(t *testing.T) {
			_, err := conn.Exec(ctx, `UPDATE scores SET status = 'not_started' WHERE scoreid = $1`, saved.ScoreID)
			require.NoError(t, err)
		}, func() error {
			_, err := mustFindScore(t, saved.ScoreID).Confirm(fx.editorCtx)
			return err
		}},
		{"Undo", func(t *testing.T) {
			_, err := (&Score{ScoreID: saved.ScoreID, FismaSystemID: fx.fismaSystemID, FunctionOptionID: fx.functionOptionID, DataCallID: fx.dataCallID}).Save(fx.editorCtx)
			require.NoError(t, err)
		}, func() error {
			stored := mustFindScore(t, saved.ScoreID)
			history, err := FindScoreRevisions(fx.editorCtx, stored, writePolicy)
			if err != nil {
				return err
			}
			head := history.Head.RevisionID
			_, err = UndoScoreRevision(fx.editorCtx, stored, &head)
			return err
		}},
	}
	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			if p.setup != nil {
				p.setup(t)
			}
			repin, err := db.Conn(ctx)
			require.NoError(t, err)
			defer repin.Release()
			repinTx, err := repin.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = repinTx.Rollback(ctx) }()
			_, err = repinTx.Exec(ctx, `SELECT 1 FROM datacalls WHERE datacallid = $1 FOR UPDATE`, fx.dataCallID)
			require.NoError(t, err)

			done := make(chan error, 1)
			go func() { done <- p.run() }()
			select {
			case err := <-done:
				t.Fatalf("the write finished while the data call was locked: %v", err)
			case <-time.After(300 * time.Millisecond):
			}

			probe, err := db.Conn(ctx)
			require.NoError(t, err)
			defer probe.Release()
			probeTx, err := probe.Begin(ctx)
			require.NoError(t, err)
			_, err = probeTx.Exec(ctx, `SELECT 1 FROM scores WHERE scoreid = $1 FOR UPDATE NOWAIT`, saved.ScoreID)
			assert.NoError(t, err, "the write must be waiting on the data call, not holding the score row")
			require.NoError(t, probeTx.Rollback(ctx))

			require.NoError(t, repinTx.Rollback(ctx))
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(10 * time.Second):
				t.Fatal("the write never finished after the data call was released")
			}
		})
	}
}

// newDraftQuestion opens a committed draft version holding one new question in
// pillarID, for tests that write catalog content through the model, which
// only accepts writes to a draft since ztmf-misc#396. The integration_test_
// label lets purgeIntegrationTestVersions remove it.
func newDraftQuestion(t *testing.T, pillarID int32) int32 {
	t.Helper()
	purgeIntegrationTestVersions(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var questionID int32
	require.NoError(t, conn.QueryRow(ctx, `
		WITH d AS (INSERT INTO questionnaire_versions (label) VALUES ($1) RETURNING versionid)
		INSERT INTO questions (question, notesprompt, pillarid, versionid)
		SELECT 'draft fixture', 'prompt', $2, versionid FROM d RETURNING questionid`,
		integrationTestPrefix+"draftq", pillarID).Scan(&questionID))
	t.Cleanup(func() { purgeIntegrationTestVersions(t) })
	return questionID
}

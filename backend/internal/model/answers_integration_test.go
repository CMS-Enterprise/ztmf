package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindAnswersOpDivScopeIntegration pins the OpDiv scoping added for
// ztmf-misc#267 and #268 against the real SQL: an OpDiv-restricted export sees
// only systems in the granted OpDivs, an empty grant fails closed to no rows,
// and an fsids list naming a system outside the grant does not leak it.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindAnswersOpDivScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// A data call with scored systems in at least two distinct OpDivs, plus the
	// two OpDiv ids to compare. Discovered rather than hardcoded so the test
	// tracks whatever the seed provides.
	var call, opdivA, opdivB int32
	err = conn.QueryRow(ctx, `
		WITH by_call AS (
			SELECT s.datacallid, fs.opdiv_id, COUNT(DISTINCT fs.fismasystemid) n
			FROM scores s
			JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
			WHERE fs.opdiv_id IS NOT NULL
			GROUP BY s.datacallid, fs.opdiv_id
		)
		SELECT a.datacallid, a.opdiv_id, b.opdiv_id
		FROM by_call a
		JOIN by_call b ON b.datacallid = a.datacallid AND b.opdiv_id > a.opdiv_id
		ORDER BY a.datacallid, a.opdiv_id, b.opdiv_id
		LIMIT 1
	`).Scan(&call, &opdivA, &opdivB)
	if err != nil {
		t.Skip("seed has no data call with scored systems in two distinct OpDivs")
	}

	opdivOf := func(t *testing.T, answers []*Answer) map[int32]int32 {
		t.Helper()
		ids := map[int32]bool{}
		for _, a := range answers {
			ids[a.FismaSystemID] = true
		}
		result := map[int32]int32{}
		for id := range ids {
			var opdiv int32
			require.NoError(t, conn.QueryRow(ctx,
				`SELECT opdiv_id FROM fismasystems WHERE fismasystemid = $1`, id).Scan(&opdiv))
			result[id] = opdiv
		}
		return result
	}

	t.Run("RestrictedToGrantSeesOnlyThatOpDiv", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID: call,
			OpDivScope: OpDivScope{RestrictToOpDivIDs: true, OpDivIDs: []int32{opdivA}},
		})
		require.NoError(t, err)
		require.NotEmpty(t, answers, "the granted OpDiv has scored systems, so the export must not be empty")
		for id, opdiv := range opdivOf(t, answers) {
			assert.Equal(t, opdivA, opdiv, "system %d is outside the granted OpDiv but was exported", id)
		}
	})

	t.Run("EmptyGrantFailsClosed", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID: call,
			OpDivScope: OpDivScope{RestrictToOpDivIDs: true, OpDivIDs: nil},
		})
		require.NoError(t, err)
		assert.Empty(t, answers, "an OpDiv-restricted caller with no grants must get no rows, not every row")
	})

	t.Run("CrossOpDivFsidsReturnsNothing", func(t *testing.T) {
		// Ask for a system in OpDiv B while granted only OpDiv A: the two filters
		// conjoin, so the out-of-scope system is dropped rather than leaked.
		var victim int32
		err := conn.QueryRow(ctx, `
			SELECT DISTINCT fs.fismasystemid
			FROM fismasystems fs
			JOIN scores s ON s.fismasystemid = fs.fismasystemid
			WHERE fs.opdiv_id = $1 AND s.datacallid = $2
			LIMIT 1`, opdivB, call).Scan(&victim)
		require.NoError(t, err)

		answers, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID:     call,
			FismaSystemIDs: []*int32{&victim},
			OpDivScope:     OpDivScope{RestrictToOpDivIDs: true, OpDivIDs: []int32{opdivA}},
		})
		require.NoError(t, err)
		assert.Empty(t, answers, "a system in another OpDiv must not be exported even when named in fsids")
	})

	t.Run("UnrestrictedSeesBothOpDivs", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{DataCallID: call})
		require.NoError(t, err)
		seen := map[int32]bool{}
		for _, opdiv := range opdivOf(t, answers) {
			seen[opdiv] = true
		}
		assert.True(t, seen[opdivA] && seen[opdivB],
			"an unscoped caller must still see systems from both OpDivs")
	})
}

// TestFindAnswersIntegration pins the #526 behavior against the real SQL: the
// export is anchored on the applicable-function set, not on scores, so a system
// that has not answered a function still appears with a blank (nil) answer, and
// the applicable set per system matches what the questionnaire and
// /scores/progress resolve. It also pins the decommissioned-or-answered guard:
// a decommissioned system is included only for functions it actually answered,
// preserving the historical rows the scores-anchored export produced without
// padding the file with blank questionnaires for retired systems.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindAnswersIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// A never-started pair: an active system applicable to at least one function
	// that has no score rows in some data call. This is exactly the population the
	// old scores-anchored export dropped.
	var neverStartedSystem, neverStartedCall int32
	err = conn.QueryRow(ctx, `
		SELECT fs.fismasystemid, dc.datacallid
		FROM fismasystems fs
		JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		JOIN functions f ON f.datacenterenvironment = dce.scoring_key
		CROSS JOIN datacalls dc
		WHERE fs.decommissioned = FALSE
		  AND NOT EXISTS (SELECT 1 FROM scores s WHERE s.fismasystemid = fs.fismasystemid AND s.datacallid = dc.datacallid)
		GROUP BY fs.fismasystemid, dc.datacallid
		ORDER BY fs.fismasystemid, dc.datacallid
		LIMIT 1
	`).Scan(&neverStartedSystem, &neverStartedCall)
	require.NoError(t, err, "seed should contain at least one active never-started (system, datacall) pair")

	// How many functions apply to that system, resolved the same way the
	// questionnaire and /scores/progress resolve them.
	var applicableCount int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(DISTINCT f.functionid)
		FROM fismasystems fs
		JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		JOIN functions f ON f.datacenterenvironment = dce.scoring_key
		JOIN questions q ON q.questionid = f.questionid
		JOIN pillars p ON p.pillarid = q.pillarid
		WHERE fs.fismasystemid = $1
	`, neverStartedSystem).Scan(&applicableCount)
	require.NoError(t, err)
	require.Greater(t, applicableCount, 0)

	// The export for that call, scoped to the never-started system.
	rows, err := FindAnswers(ctx, FindAnswersInput{
		DataCallID:     neverStartedCall,
		FismaSystemIDs: []*int32{&neverStartedSystem},
	})
	require.NoError(t, err)

	assert.Len(t, rows, applicableCount,
		"a never-started system should contribute one row per applicable function, matching /scores/progress")
	for _, r := range rows {
		assert.Equal(t, neverStartedSystem, r.FismaSystemID)
		assert.Nil(t, r.Score, "a never-started function must carry a nil score, not a coalesced zero")
		assert.Nil(t, r.OptionName, "a never-started function must carry a nil option")
		assert.Nil(t, r.Notes, "a never-started function must carry nil notes")
	}

	// Regression guard: an answered system in the same call still exports its
	// selected options. Find one that has scores in this call.
	var answeredSystem int32
	err = conn.QueryRow(ctx, `
		SELECT DISTINCT s.fismasystemid
		FROM scores s
		JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		WHERE s.datacallid = $1 AND fs.decommissioned = FALSE
		LIMIT 1
	`, neverStartedCall).Scan(&answeredSystem)
	if err == nil {
		answeredRows, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID:     neverStartedCall,
			FismaSystemIDs: []*int32{&answeredSystem},
		})
		require.NoError(t, err)
		var withAnswer int
		for _, r := range answeredRows {
			if r.Score != nil {
				withAnswer++
				assert.NotNil(t, r.OptionName, "an answered row should carry its selected option")
			}
		}
		assert.Greater(t, withAnswer, 0, "an answered system should still export its answered functions")
	}

	// Orphaned answers (ztmf-misc#384, reversing #528's widening): an active
	// system whose environment changed mid-cycle holds answers on functions its
	// current environment no longer includes. Scoring's expected CTE never
	// enumerates those functions, so they contribute nothing to the dashboard and
	// must not appear in the export either - otherwise the workbook shows two rows
	// per question and only one of them is the one being scored.
	//
	// Systems under a reduced-pillar rule are excluded from the pick so the
	// expected count below does not have to re-derive that filter.
	var orphanSystem, orphanCall int32
	var orphanAnswered, applicableAnswered int
	err = conn.QueryRow(ctx, `
		WITH sysscores AS (
			SELECT s.fismasystemid, s.datacallid,
			       EXISTS (
			           SELECT 1 FROM datacenterenvironments dce
			           JOIN functions af ON af.datacenterenvironment = dce.scoring_key
			           WHERE dce.datacenterenvironment = fs.datacenterenvironment
			             AND af.functionid = fo.functionid
			       ) AS applicable
			FROM scores s
			JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
			JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
			JOIN datacenterenvironments sdce ON sdce.datacenterenvironment = fs.datacenterenvironment
			WHERE fs.decommissioned = FALSE
			  AND NOT EXISTS (SELECT 1 FROM reducedpillarscopes rps WHERE rps.scoring_key = sdce.scoring_key)
		)
		SELECT fismasystemid, datacallid,
		       COUNT(*) FILTER (WHERE NOT applicable) AS orphaned,
		       COUNT(*) FILTER (WHERE applicable)     AS applicable_answered
		FROM sysscores
		GROUP BY fismasystemid, datacallid
		HAVING COUNT(*) FILTER (WHERE NOT applicable) > 0
		ORDER BY fismasystemid, datacallid
		LIMIT 1
	`).Scan(&orphanSystem, &orphanCall, &orphanAnswered, &applicableAnswered)
	if err == nil {
		orphanRows, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID:     orphanCall,
			FismaSystemIDs: []*int32{&orphanSystem},
		})
		require.NoError(t, err)
		var answered int
		for _, r := range orphanRows {
			if r.Score != nil {
				answered++
			}
		}
		assert.Equal(t, applicableAnswered, answered,
			"the export must carry only answers on currently applicable functions; %d orphaned answer(s) must not appear",
			orphanAnswered)
	} else {
		t.Log("no active system with orphaned answers in seed; skipping the orphaned-answer assertion")
	}

	// Decommissioned-or-answered guard: a decommissioned system is included only
	// for functions it actually answered. Find a decommissioned system with
	// scores in some call and assert its export equals its answered-function
	// count, all rows non-nil.
	var decomSystem, decomCall int32
	err = conn.QueryRow(ctx, `
		SELECT fs.fismasystemid, s.datacallid
		FROM fismasystems fs
		JOIN scores s ON s.fismasystemid = fs.fismasystemid
		WHERE fs.decommissioned = TRUE
		GROUP BY fs.fismasystemid, s.datacallid
		ORDER BY fs.fismasystemid, s.datacallid
		LIMIT 1
	`).Scan(&decomSystem, &decomCall)
	if err == nil {
		// Counted the way the export now admits rows (ztmf-misc#384): answers on
		// applicable functions, plus every answer when the system has no applicable
		// catalog at all - the usual case once an environment is retired to the
		// DECOMMISSIONED marker, and the reason the answered branch survives.
		// COUNT(*), not COUNT(DISTINCT functionid), because a duplicate score row on
		// one function (ztmf#491) is its own exported row.
		var answeredFns int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM scores s
			JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
			WHERE s.fismasystemid = $1 AND s.datacallid = $2
			  AND (
			    EXISTS (
			        SELECT 1 FROM fismasystems fs
			        JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
			        JOIN functions af ON af.datacenterenvironment = dce.scoring_key
			        WHERE fs.fismasystemid = s.fismasystemid AND af.functionid = fo.functionid)
			    OR NOT EXISTS (
			        SELECT 1 FROM fismasystems fs
			        JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
			        JOIN functions af ON af.datacenterenvironment = dce.scoring_key
			        WHERE fs.fismasystemid = s.fismasystemid)
			  )
		`, decomSystem, decomCall).Scan(&answeredFns)
		require.NoError(t, err)

		decomRows, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID:     decomCall,
			FismaSystemIDs: []*int32{&decomSystem},
		})
		require.NoError(t, err)
		assert.Len(t, decomRows, answeredFns,
			"a decommissioned system should export only its answered functions, not a full blank questionnaire")
		for _, r := range decomRows {
			assert.NotNil(t, r.Score, "a decommissioned system's exported rows must all be answered rows")
		}
	} else {
		t.Log("no decommissioned system with scores in seed; skipping the decommissioned-guard assertion")
	}

	// Decommissioned AND still mapped AND holding orphaned answers - the one
	// combination the two blocks above can each miss. The fallback branch does not
	// fire (an applicable catalog exists), so only answers on applicable functions
	// survive, and a system whose answers are all orphaned exports nothing at all
	// rather than its old catalog (ztmf-misc#384; the rows stay in scores).
	var mappedDecomSystem, mappedDecomCall int32
	var mappedDecomOrphans, mappedDecomKept int
	err = conn.QueryRow(ctx, `
		WITH applicable AS (
			SELECT fs.fismasystemid, f.functionid
			FROM fismasystems fs
			JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
			JOIN functions f ON f.datacenterenvironment = dce.scoring_key
		)
		SELECT s.fismasystemid, s.datacallid,
		       COUNT(*) FILTER (WHERE NOT EXISTS (
		           SELECT 1 FROM applicable a
		           WHERE a.fismasystemid = s.fismasystemid AND a.functionid = fo.functionid)) AS orphaned,
		       COUNT(*) FILTER (WHERE EXISTS (
		           SELECT 1 FROM applicable a
		           WHERE a.fismasystemid = s.fismasystemid AND a.functionid = fo.functionid)) AS kept
		FROM scores s
		JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		WHERE fs.decommissioned = TRUE
		  AND EXISTS (SELECT 1 FROM applicable a WHERE a.fismasystemid = fs.fismasystemid)
		GROUP BY s.fismasystemid, s.datacallid
		HAVING COUNT(*) FILTER (WHERE NOT EXISTS (
		           SELECT 1 FROM applicable a
		           WHERE a.fismasystemid = s.fismasystemid AND a.functionid = fo.functionid)) > 0
		ORDER BY s.fismasystemid, s.datacallid
		LIMIT 1
	`).Scan(&mappedDecomSystem, &mappedDecomCall, &mappedDecomOrphans, &mappedDecomKept)
	if err == nil {
		rows, err := FindAnswers(ctx, FindAnswersInput{
			DataCallID:     mappedDecomCall,
			FismaSystemIDs: []*int32{&mappedDecomSystem},
		})
		require.NoError(t, err)
		assert.Len(t, rows, mappedDecomKept,
			"a mapped decommissioned system must export only its %d answers on applicable functions, not the %d orphaned ones",
			mappedDecomKept, mappedDecomOrphans)
		for _, r := range rows {
			assert.NotNil(t, r.Score, "every exported row for a decommissioned system must be an answered row")
		}
	} else {
		t.Log("no mapped decommissioned system with orphaned answers in seed; skipping that assertion")
	}
}

// TestFindAnswersSaaSPillarScopeIntegration pins the export half of
// ztmf-misc#289: 25 questions for SaaS from FY26 on, 40 on earlier cycles.
//
// Asserting that NO exported row carries an excluded pillar covers both branches
// of the applicable-OR-answered join (#528) at once - filtering only one leaves
// carried-forward systems exporting 40 while fresh ones export 25.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindAnswersSaaSPillarScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	fy26, prior, ok := reducedScopeAnchorAndPriorDataCalls(ctx, t)
	if !ok {
		t.Skip("database has no seeded reduced-scope rule with an earlier cycle to compare against")
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// Every SaaS system, plus one carrying answers on an excluded pillar in FY26 -
	// the case the OR branch would readmit.
	saas := map[int32]bool{}
	rows, err := conn.Query(ctx, `
		SELECT fs.fismasystemid
		FROM fismasystems fs
		JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		WHERE dce.scoring_key = 'SaaS'`)
	require.NoError(t, err)
	for rows.Next() {
		var id int32
		require.NoError(t, rows.Scan(&id))
		saas[id] = true
	}
	rows.Close()
	if len(saas) == 0 {
		t.Skip("database carries no SaaS systems")
	}

	var carriedForward int32
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid
		FROM scores s
		JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		JOIN functions f ON f.functionid = fo.functionid
		JOIN questions q ON q.questionid = f.questionid
		JOIN pillars p ON p.pillarid = q.pillarid
		JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		WHERE s.datacallid = $1 AND dce.scoring_key = 'SaaS'
		  AND p.pillar IN ('Devices', 'Applications')
		LIMIT 1
	`, fy26).Scan(&carriedForward)
	haveCarried := err == nil

	t.Run("FY26ExcludesTheReducedPillars", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{DataCallID: fy26})
		require.NoError(t, err)
		require.NotEmpty(t, answers)

		seenSaaS := map[int32]bool{}
		for _, a := range answers {
			if !saas[a.FismaSystemID] {
				continue
			}
			seenSaaS[a.FismaSystemID] = true
			assert.NotContains(t, []string{"Devices", "Applications"}, a.Pillar,
				"system %d exported an excluded pillar (%s) on the FY26 call", a.FismaSystemID, a.Pillar)
		}
		assert.NotEmpty(t, seenSaaS, "SaaS systems must still appear in the export, just with fewer questions")

		// Row counts per system are not a useful assertion: the #528
		// applicable-or-answered join legitimately doubles them for a system
		// carrying answers from another edition. The surviving pillar SET is, and
		// it also catches over-pruning inside the in-scope pillars.
		var inScope, expectedInScope int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(DISTINCT p.pillar)
			FROM pillars p
			WHERE p.pillar NOT IN ('Devices', 'Applications')
			  AND EXISTS (
			      SELECT 1 FROM questions q
			      JOIN functions f ON f.questionid = q.questionid
			      WHERE q.pillarid = p.pillarid AND f.datacenterenvironment = 'SaaS')
		`).Scan(&expectedInScope)
		require.NoError(t, err)

		pillars := map[string]bool{}
		for _, a := range answers {
			if saas[a.FismaSystemID] {
				pillars[a.Pillar] = true
			}
		}
		inScope = len(pillars)
		assert.Equal(t, expectedInScope, inScope,
			"every in-scope pillar must still be exported, not just the excluded ones dropped")

		if haveCarried {
			assert.True(t, seenSaaS[carriedForward],
				"the system with carried-forward excluded answers must still export its in-scope questions")
		}
	})

	t.Run("PriorCycleStillExportsEveryPillar", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{DataCallID: prior})
		require.NoError(t, err)

		excluded := 0
		for _, a := range answers {
			if saas[a.FismaSystemID] && (a.Pillar == "Devices" || a.Pillar == "Applications") {
				excluded++
			}
		}
		assert.Greater(t, excluded, 0,
			"cycles earlier than FY26 collected the full set and must keep exporting it")
	})

	t.Run("NonSaaSSystemsAreUntouchedOnFY26", func(t *testing.T) {
		answers, err := FindAnswers(ctx, FindAnswersInput{DataCallID: fy26})
		require.NoError(t, err)

		excluded := 0
		for _, a := range answers {
			if !saas[a.FismaSystemID] && (a.Pillar == "Devices" || a.Pillar == "Applications") {
				excluded++
			}
		}
		assert.Greater(t, excluded, 0,
			"the scope is SaaS-only; other environments still export both pillars")
	})
}

// TestFindAnswersScoringParityIntegration is the regression guard for
// ztmf-misc#384: the export must emit exactly the rows the dashboard scores,
// never a second row for a question the system answered under a previous
// environment. BHP-TAP exported 50 rows for a 25-question SaaS catalog because
// the functions join also admitted answered-but-no-longer-applicable functions.
//
// Expected row count per system is derived from the same applicable set scoring
// uses, allowing for more than one row where a duplicate score row exists on a
// single function (ztmf#491, which this ticket does not fix).
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindAnswersScoringParityIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// The busiest data call, so the comparison covers real answered systems.
	var dataCallID int32
	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM scores GROUP BY datacallid ORDER BY COUNT(*) DESC LIMIT 1
	`).Scan(&dataCallID)
	require.NoError(t, err, "seed should contain at least one data call with scores")

	// Bounded sample, systems holding orphaned answers first: those are the ones
	// that regress, and a whole-call export here costs minutes.
	rows, err := conn.Query(ctx, `
		WITH applicable AS (
			SELECT fs.fismasystemid, f.functionid
			FROM fismasystems fs
			JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
			JOIN functions f ON f.datacenterenvironment = dce.scoring_key
			JOIN questions q ON q.questionid = f.questionid
			JOIN pillars p ON p.pillarid = q.pillarid
			WHERE fs.decommissioned = FALSE
			  AND `+reducedPillarScopeSQL("dce.scoring_key", "p.pillar", "$1")+`
		),
		orphaned AS (
			SELECT DISTINCT s.fismasystemid
			FROM scores s
			JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
			WHERE s.datacallid = $1
			  AND NOT EXISTS (
			      SELECT 1 FROM applicable a
			      WHERE a.fismasystemid = s.fismasystemid AND a.functionid = fo.functionid)
		),
		sample AS (
			SELECT DISTINCT a.fismasystemid,
			       (a.fismasystemid IN (SELECT fismasystemid FROM orphaned)) AS has_orphans
			FROM applicable a
			ORDER BY has_orphans DESC, a.fismasystemid
			LIMIT 25
		)
		SELECT sm.fismasystemid, sm.has_orphans,
		       SUM(GREATEST(1, (
		           SELECT COUNT(*) FROM scores s
		           JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		           WHERE s.fismasystemid = sm.fismasystemid
		             AND s.datacallid = $1
		             AND fo.functionid = a.functionid
		       )))::int AS expected_rows
		FROM sample sm
		JOIN applicable a ON a.fismasystemid = sm.fismasystemid
		GROUP BY sm.fismasystemid, sm.has_orphans
	`, dataCallID)
	require.NoError(t, err)

	expected := map[int32]int{}
	var fsids []*int32
	var withOrphans int
	for rows.Next() {
		var sysID int32
		var hasOrphans bool
		var n int
		require.NoError(t, rows.Scan(&sysID, &hasOrphans, &n))
		expected[sysID] = n
		id := sysID
		fsids = append(fsids, &id)
		if hasOrphans {
			withOrphans++
		}
	}
	require.NoError(t, rows.Err())
	rows.Close()
	require.NotEmpty(t, expected, "seed should contain at least one active system with an applicable catalog")
	if withOrphans == 0 {
		t.Log("no sampled system holds orphaned answers; the assertion still pins the grain but not the #384 regression")
	}

	answers, err := FindAnswers(ctx, FindAnswersInput{DataCallID: dataCallID, FismaSystemIDs: fsids})
	require.NoError(t, err)

	actual := map[int32]int{}
	for _, a := range answers {
		actual[a.FismaSystemID]++
	}

	// Absence is a failure, not a skip: applicable already restricts to active
	// systems with a mapped catalog and at least one in-scope function, so the
	// applicable branch of the functions join must fire and every one of these
	// systems must export at least one row. Dropping a system outright is the
	// regression this test exists to catch, and it is the likeliest way the
	// no-applicable-catalog fallback could go wrong.
	for sysID, want := range expected {
		got, ok := actual[sysID]
		require.Truef(t, ok, "system %d has %d applicable functions but exported no rows", sysID, want)
		assert.Equal(t, want, got,
			"system %d exports %d rows but scoring enumerates %d - the export is showing answers the dashboard does not count",
			sysID, got, want)
	}
}

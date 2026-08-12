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

	// Orphaned answers (the reason FindAnswers keys off applicable-OR-answered,
	// not applicable alone): an active system whose answers reference functions no
	// longer applicable to its current environment must still export those answers,
	// not vanish. Find an active system with scores whose applicable set does not
	// cover every answered function.
	var orphanSystem, orphanCall int32
	var orphanAnswered int
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.datacallid, COUNT(DISTINCT fo.functionid) AS answered
		FROM scores s
		JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		WHERE fs.decommissioned = FALSE
		  AND NOT EXISTS (
		      SELECT 1 FROM datacenterenvironments dce
		      JOIN functions af ON af.datacenterenvironment = dce.scoring_key
		      WHERE dce.datacenterenvironment = fs.datacenterenvironment AND af.functionid = fo.functionid)
		GROUP BY s.fismasystemid, s.datacallid
		ORDER BY s.fismasystemid, s.datacallid
		LIMIT 1
	`).Scan(&orphanSystem, &orphanCall, &orphanAnswered)
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
		assert.GreaterOrEqual(t, answered, orphanAnswered,
			"an active system's orphaned answers (functions no longer applicable) must still export, not drop")
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
		var answeredFns int
		err = conn.QueryRow(ctx, `
			SELECT COUNT(DISTINCT fo.functionid)
			FROM scores s
			JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
			WHERE s.fismasystemid = $1 AND s.datacallid = $2
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

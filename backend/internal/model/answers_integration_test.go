package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

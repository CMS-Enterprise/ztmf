package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindQuestionsByFismaSystemReducedScopeIntegration pins ztmf#545's
// questionnaire contract in real Postgres: with a data call supplied, questions
// on a pillar the seeded rule excludes are omitted from the anchor cycle
// onward; without one, or for cycles before the anchor, the environment's full
// catalog is returned unchanged (the pre-#545 contract the frontend's own
// pillar filter still expects).
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindQuestionsByFismaSystemReducedScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	anchor, prior, ok := reducedScopeAnchorAndPriorDataCalls(ctx, t)
	if !ok {
		t.Skip("database has no seeded reduced-scope rule with an earlier cycle to compare against")
	}

	systemID, found := findSystemIDByAcronym(ctx, t, "MOCK-SAAS")
	if !found {
		t.Skip("fixture system MOCK-SAAS not seeded")
	}

	questionsFor := func(t *testing.T, systemID int32, dataCallID *int32) []*Question {
		t.Helper()
		qs, err := FindQuestionsByFismaSystem(ctx, systemID, FindQuestionsByFismaSystemInput{DataCallID: dataCallID})
		require.NoError(t, err)
		require.NotEmpty(t, qs)
		return qs
	}

	countOnExcludedPillars := func(qs []*Question) int {
		n := 0
		for _, q := range qs {
			if q.Pillar.Pillar == "Devices" || q.Pillar.Pillar == "Applications" {
				n++
			}
		}
		return n
	}

	baseline := questionsFor(t, systemID, nil)
	require.Positive(t, countOnExcludedPillars(baseline),
		"the fixture catalog must have questions on the excluded pillars, or this test proves nothing")

	t.Run("NoDataCallKeepsTheFullCatalog", func(t *testing.T) {
		// baseline above IS the assertion: the nil path returned excluded-pillar
		// questions, so the endpoint without the param is byte-compatible with
		// the pre-#545 contract.
		assert.Positive(t, countOnExcludedPillars(baseline))
	})

	t.Run("AnchorCycleServesTheReducedSet", func(t *testing.T) {
		reduced := questionsFor(t, systemID, &anchor)
		assert.Zero(t, countOnExcludedPillars(reduced),
			"questions on an excluded pillar must not be served for the anchor cycle")
		assert.Len(t, reduced, len(baseline)-countOnExcludedPillars(baseline),
			"exactly the excluded pillars' questions drop; everything else stays")
	})

	t.Run("PriorCycleKeepsTheFullCatalog", func(t *testing.T) {
		full := questionsFor(t, systemID, &prior)
		assert.Len(t, full, len(baseline),
			"cycles deadlined before the anchor keep the full question set")
		assert.Positive(t, countOnExcludedPillars(full))
	})

	t.Run("OtherEnvironmentsUnreducedOnTheAnchorCycle", func(t *testing.T) {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Release()

		// Any system scored under a key with no rule row serves as the control.
		var controlID int32
		err = conn.QueryRow(ctx, `
			SELECT fs.fismasystemid
			FROM fismasystems fs
			INNER JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
			WHERE dce.scoring_key IS NOT NULL
			  AND dce.scoring_key NOT IN (SELECT scoring_key FROM reducedpillarscopes)
			ORDER BY fs.fismasystemid LIMIT 1
		`).Scan(&controlID)
		if err != nil {
			t.Skip("no non-reduced system to use as a control")
		}

		ctrlBaseline := questionsFor(t, controlID, nil)
		ctrlOnAnchor := questionsFor(t, controlID, &anchor)
		assert.Len(t, ctrlOnAnchor, len(ctrlBaseline),
			"a system without a rule row keeps its full catalog on the anchor cycle")
	})
}

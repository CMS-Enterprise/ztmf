package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findSystemIDByAcronym resolves a seeded fixture system. Discovery + skip
// rather than a hardcoded id, matching saasScopeFixture: the test pins the
// rule, not the seed file's numbering.
func findSystemIDByAcronym(ctx context.Context, t *testing.T, acronym string) (int32, bool) {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var id int32
	if err := conn.QueryRow(ctx,
		`SELECT fismasystemid FROM fismasystems WHERE fismaacronym = $1`, acronym,
	).Scan(&id); err != nil {
		return 0, false
	}
	return id, true
}

// TestFindScoresAggregateReducedScopeIntegration pins ztmf#545's scoring
// contract in real Postgres: from the seeded anchor cycle onward a reduced
// system carries no pillar score for the excluded pillars and its system score
// averages over the pillars that remain, while cycles before the anchor keep
// every pillar. The fixture deliberately carries FY26 answers on the excluded
// pillars (carried forward as 'not_started'), so this also proves those
// answers stop counting rather than resurfacing through the answers CTE.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindScoresAggregateReducedScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	anchor, prior, ok := reducedScopeAnchorAndPriorDataCalls(ctx, t)
	if !ok {
		t.Skip("database has no seeded reduced-scope rule with an earlier cycle to compare against")
	}

	aggregateFor := func(t *testing.T, systemID, dataCallID int32) *ScoreAggregate {
		t.Helper()
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{
			DataCallID:     &dataCallID,
			FismaSystemID:  &systemID,
			IncludePillars: boolPtr(true),
		})
		require.NoError(t, err)
		require.Len(t, aggs, 1, "one aggregate per (datacall, system) pair")
		require.NotEmpty(t, aggs[0].PillarScores)
		return aggs[0]
	}

	pillarNames := func(a *ScoreAggregate) []string {
		names := make([]string, 0, len(a.PillarScores))
		for _, p := range a.PillarScores {
			names = append(names, p.Pillar)
		}
		return names
	}

	// The CMS twin asserts the ztmf-misc#289 decision (2026-08-11): CMS follows
	// every other OpDiv, so both fixtures must behave identically.
	for name, acronym := range map[string]string{
		"NonCMS":                        "MOCK-SAAS",
		"CMSReducedLikeEveryOtherOpDiv": "MOCK-SAAS-CMS",
	} {
		t.Run(name, func(t *testing.T) {
			systemID, found := findSystemIDByAcronym(ctx, t, acronym)
			if !found {
				t.Skipf("fixture system %s not seeded", acronym)
			}

			onPrior := aggregateFor(t, systemID, prior)
			assert.Contains(t, pillarNames(onPrior), "Devices",
				"cycles before the anchor keep every pillar")
			assert.Contains(t, pillarNames(onPrior), "Applications")

			onAnchor := aggregateFor(t, systemID, anchor)
			assert.NotContains(t, pillarNames(onAnchor), "Devices",
				"an excluded pillar must not carry a score from the anchor cycle onward, even with carried-forward answers present")
			assert.NotContains(t, pillarNames(onAnchor), "Applications")
			assert.Len(t, onAnchor.PillarScores, len(onPrior.PillarScores)-2,
				"exactly the two excluded pillars drop; the rest remain")

			// The system score is the average of the pillars present - the
			// existing divisor-follows-the-pillar-count contract, now over 4.
			sum := 0.0
			for _, p := range onAnchor.PillarScores {
				sum += p.Score
			}
			assert.InDelta(t, sum/float64(len(onAnchor.PillarScores)), onAnchor.SystemScore, 1e-9,
				"the system score must average only the in-scope pillars")
			assert.Equal(t, Tier(onAnchor.SystemScore), onAnchor.SystemTier)
		})
	}
}

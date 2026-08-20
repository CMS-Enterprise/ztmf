package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReducedPillarScopeNullSafetyIntegration evaluates the predicate in real
// Postgres, because its correctness rests on three-valued logic that no string
// assertion can pin: a NULL scoring key must yield TRUE (row kept), never NULL,
// which a top-level WHERE would read as "drop". NOT EXISTS provides that by
// construction, and this test is what fails if a refactor trades it away.
//
// The regression this guards: scoring_key is nullable (DECOMMISSIONED maps to
// NULL) and FindAnswers deliberately still exports decommissioned systems that
// carry scores, so a NULL key mishandled here silently removes excluded-pillar
// rows from a compliance export - on a non-SaaS system.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestReducedPillarScopeNullSafetyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	anchor, prior, ok := reducedScopeAnchorAndPriorDataCalls(ctx, t)
	if !ok {
		t.Skip("database has no seeded reduced-scope rule with an earlier cycle to compare against")
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// kept == true means the row survives, i.e. the full question set.
	kept := func(t *testing.T, scoringKeyExpr, pillarExpr string, dataCallID int32) bool {
		t.Helper()
		var result *bool
		require.NoError(t, conn.QueryRow(ctx,
			`SELECT `+reducedPillarScopeSQL(scoringKeyExpr, pillarExpr, "$1"), dataCallID,
		).Scan(&result))
		require.NotNil(t, result, "the predicate must never evaluate to NULL: a WHERE would drop the row and a JOIN ... ON would too")
		return *result
	}

	t.Run("SeededRuleRowsPresent", func(t *testing.T) {
		// Pins the seed wiring end to end: migration 0057 creates the table, and
		// either its name-lookup (deployed envs) or the populate seed (local/test
		// envs) provides exactly the two SaaS rows the rest of this test relies on.
		var n int
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM reducedpillarscopes rps
			INNER JOIN pillars p ON p.pillarid = rps.pillarid
			WHERE rps.scoring_key = 'SaaS' AND p.pillar IN ('Devices', 'Applications')
		`).Scan(&n))
		assert.Equal(t, 2, n, "the SaaS rule is two rows: Devices and Applications")
	})

	t.Run("NullScoringKeyKeepsEveryPillar", func(t *testing.T) {
		assert.True(t, kept(t, "NULL::varchar", "'Devices'", anchor),
			"a system with no scoring key is not SaaS and must keep Devices")
		assert.True(t, kept(t, "NULL::varchar", "'Applications'", anchor),
			"a system with no scoring key is not SaaS and must keep Applications")
	})

	t.Run("SaaSDropsOnlyTheExcludedPillars", func(t *testing.T) {
		assert.False(t, kept(t, "'SaaS'", "'Devices'", anchor))
		assert.False(t, kept(t, "'SaaS'", "'Applications'", anchor))
		assert.True(t, kept(t, "'SaaS'", "'Identity'", anchor))
		assert.True(t, kept(t, "'SaaS'", "'CrossCutting'", anchor))
	})

	t.Run("OtherEnvironmentsAndPriorCyclesKeepEverything", func(t *testing.T) {
		assert.True(t, kept(t, "'AWS'", "'Devices'", anchor),
			"only environments with a rule row are reduced")
		assert.True(t, kept(t, "'SaaS'", "'Devices'", prior),
			"cycles deadlined before the anchor keep the full question set")
	})

	t.Run("UnknownDataCallFallsBackToTheFullSet", func(t *testing.T) {
		assert.True(t, kept(t, "'SaaS'", "'Devices'", -1),
			"an unresolvable cycle must fail open to the full set, never NULL")
	})
}

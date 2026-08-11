package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaaSPillarScopeNullSafetyIntegration evaluates the predicate in real
// Postgres, because its correctness rests on three-valued logic that no
// string assertion can pin: a NULL scoring key must yield FALSE (row kept), not
// NULL, which a top-level WHERE would read as "drop".
//
// The regression this guards: scoring_key is nullable (DECOMMISSIONED maps to
// NULL) and FindAnswers deliberately still exports decommissioned systems that
// carry scores, so a NULL key used to silently remove 15 Devices/Applications
// rows from a compliance export - on a non-SaaS system.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestSaaSPillarScopeNullSafetyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	fy26, prior, ok := fy26AndPriorDataCalls(ctx, t)
	if !ok {
		t.Skip("database has no FY26 call with an earlier cycle to compare against")
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// kept == true means the row survives, i.e. the full 40.
	kept := func(t *testing.T, scoringKeyExpr, pillarExpr string, dataCallID int32) bool {
		t.Helper()
		var result *bool
		require.NoError(t, conn.QueryRow(ctx,
			`SELECT `+saasPillarScopeSQL(scoringKeyExpr, pillarExpr, "$1"), dataCallID,
		).Scan(&result))
		require.NotNil(t, result, "the predicate must never evaluate to NULL: a WHERE would drop the row and a JOIN ... ON would too")
		return *result
	}

	t.Run("NullScoringKeyKeepsEveryPillar", func(t *testing.T) {
		assert.True(t, kept(t, "NULL::varchar", "'Devices'", fy26),
			"a system with no scoring key is not SaaS and must keep Devices")
		assert.True(t, kept(t, "NULL::varchar", "'Applications'", fy26),
			"a system with no scoring key is not SaaS and must keep Applications")
	})

	t.Run("SaaSDropsOnlyTheExcludedPillars", func(t *testing.T) {
		assert.False(t, kept(t, "'SaaS'", "'Devices'", fy26))
		assert.False(t, kept(t, "'SaaS'", "'Applications'", fy26))
		assert.True(t, kept(t, "'SaaS'", "'Identity'", fy26))
		assert.True(t, kept(t, "'SaaS'", "'CrossCutting'", fy26))
	})

	t.Run("OtherEnvironmentsAndPriorCyclesKeepEverything", func(t *testing.T) {
		assert.True(t, kept(t, "'AWS'", "'Devices'", fy26),
			"the scope is SaaS-only")
		assert.True(t, kept(t, "'SaaS'", "'Devices'", prior),
			"cycles earlier than FY26 keep the full question set")
	})

	t.Run("UnknownDataCallFallsBackToTheFullSet", func(t *testing.T) {
		assert.True(t, kept(t, "'SaaS'", "'Devices'", -1),
			"an unresolvable cycle must fail open to 40, never NULL")
	})
}

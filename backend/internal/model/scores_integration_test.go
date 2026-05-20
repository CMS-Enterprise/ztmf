package model

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validTiers is the authoritative tier-name enum the API serves. Tests
// assert membership rather than exact values where the underlying data
// is dynamic, and exact values where a transaction-rolled-back fixture
// pins the input.
var validTiers = map[string]bool{
	"Optimal":      true,
	"Advanced":     true,
	"Initial":      true,
	"Traditional":  true,
	"Not Assessed": true,
}

// integrationTestPrefix is the marker every test-synthesized datacall
// carries in its name. Used to identify rows for cleanup, including a
// defensive sweep at test entry that catches anything a previous
// interrupted run left behind. Keep it underscore-separated so a LIKE
// pattern is unambiguous.
const integrationTestPrefix = "integration_test_"

// purgeIntegrationTestRows wipes any datacalls (and their cascaded
// scores via FK ON DELETE CASCADE) whose name carries the integration
// test prefix. Run at the start of each integration test as a defensive
// belt-and-suspenders against cleanup that failed to run on a previous
// invocation. Cheap.
func purgeIntegrationTestRows(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("purge: db.Conn: %v", err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx,
		`DELETE FROM datacalls WHERE datacall LIKE $1`,
		integrationTestPrefix+"%",
	)
	if err != nil {
		t.Fatalf("purge: delete: %v", err)
	}
}

// TestFindScoresAggregateIntegration runs the real SQL against the
// configured database and verifies the response shape and per-row
// invariants. Catches SQL math regressions (wrong +1 shift, wrong AVG
// window, wrong COALESCE behavior) that unit tests over pure Go cannot
// see.
//
// Requires DB_* env vars (DB_ENDPOINT, DB_USER, etc.) pointing at a
// running Postgres with seeded ZTMF schema and at least one scored
// system. The dev compose stack (port 54321) satisfies this. The
// ephemeral test compose stack on port 8090 also works during the e2e
// step of make test-full. Skipped under `go test -short`.
func TestFindScoresAggregateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Close(ctx)

	t.Run("ShapeInvariants_AllAggregates", func(t *testing.T) {
		// Pull every aggregate from the DB. Don't filter — we want to
		// validate every row returned.
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{
			IncludePillars: boolPtr(true),
		})
		require.NoError(t, err)
		require.NotEmpty(t, aggs, "expected at least one aggregate from seeded data")

		for _, a := range aggs {
			assert.Greater(t, a.DataCallID, int32(0), "datacallid set")
			assert.Greater(t, a.FismaSystemID, int32(0), "fismasystemid set")
			assert.True(t, a.SystemScore >= 1.0 && a.SystemScore <= 5.0,
				"systemscore in HHS range: %v for system %d / datacall %d",
				a.SystemScore, a.FismaSystemID, a.DataCallID)
			assert.True(t, validTiers[a.SystemTier],
				"systemtier in enum: %q for system %d / datacall %d",
				a.SystemTier, a.FismaSystemID, a.DataCallID)
			// The tier predicate is the single source of truth. Either
			// SQL and Go agree, or this test fails and surfaces the
			// drift immediately.
			assert.Equal(t, Tier(a.SystemScore), a.SystemTier,
				"systemtier matches Tier(systemscore) for system %d / datacall %d",
				a.FismaSystemID, a.DataCallID)

			require.NotEmpty(t, a.PillarScores,
				"include_pillars=true must return pillar entries for system %d / datacall %d",
				a.FismaSystemID, a.DataCallID)

			for _, p := range a.PillarScores {
				assert.Greater(t, p.PillarID, int32(0))
				assert.NotEmpty(t, p.Pillar)
				assert.True(t, p.Score >= 1.0 && p.Score <= 5.0,
					"pillar %s score in HHS range: %v", p.Pillar, p.Score)
				assert.True(t, validTiers[p.Tier],
					"pillar %s tier in enum: %q", p.Pillar, p.Tier)
				assert.Equal(t, Tier(p.Score), p.Tier,
					"pillar %s tier matches Tier(score)", p.Pillar)
			}
		}
	})

	t.Run("IncludePillars_FalseOmitsBreakdown", func(t *testing.T) {
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{})
		require.NoError(t, err)
		require.NotEmpty(t, aggs)
		for _, a := range aggs {
			assert.Empty(t, a.PillarScores,
				"pillarscores must be omitted when IncludePillars is nil/false")
			assert.NotEmpty(t, a.SystemTier, "systemtier still populated")
			assert.True(t, a.SystemScore >= 1.0 && a.SystemScore <= 5.0,
				"systemscore still on HHS scale")
		}
	})

	t.Run("FilterByFismaSystem_ReturnsOnlyThatSystem", func(t *testing.T) {
		// Pick a system that has scores so the filter has something to
		// return. The first aggregate in the unfiltered list is a safe
		// pick.
		all, err := FindScoresAggregate(ctx, FindScoresInput{})
		require.NoError(t, err)
		require.NotEmpty(t, all)
		target := all[0].FismaSystemID

		filtered, err := FindScoresAggregate(ctx, FindScoresInput{
			FismaSystemID: &target,
		})
		require.NoError(t, err)
		require.NotEmpty(t, filtered)
		for _, a := range filtered {
			assert.Equal(t, target, a.FismaSystemID,
				"filter must scope to requested system")
		}
	})

	t.Run("FilterByUnscoredSystem_ReturnsEmpty", func(t *testing.T) {
		// 2_000_000 is well above any seeded ID and will never have
		// scores. Aggregate must be empty (not error, not a placeholder
		// "Not Assessed" row for a system that doesn't appear in the
		// scores table at all).
		unscored := int32(2_000_000)
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{
			FismaSystemID: &unscored,
		})
		require.NoError(t, err)
		assert.Empty(t, aggs,
			"unscored system must not appear in /scores/aggregate (frontend cross-references /fismasystems for those)")
	})
}

// TestScoreSaveValidationIntegration covers the validate() guards on
// the write path: notes length and past-deadline rules. Any synthesized
// datacalls carry the integrationTestPrefix and are removed by both a
// startup sweep and explicit cleanup using a fresh connection so a
// pre-closed test connection cannot silently swallow the delete.
func TestScoreSaveValidationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close(ctx)

	// Find a datacall whose deadline is in the past, plus one whose
	// deadline is in the future. The seeded datacalls do not guarantee
	// either, so we use the most-recent and oldest as proxies.
	var pastDataCallID, futureDataCallID int32
	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls
		WHERE deadline < NOW() ORDER BY deadline DESC LIMIT 1
	`).Scan(&pastDataCallID)
	require.NoError(t, err, "need at least one past-deadline datacall in seeded data")

	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls
		WHERE deadline > NOW() ORDER BY deadline ASC LIMIT 1
	`).Scan(&futureDataCallID)
	if err != nil {
		// If no future-deadline datacall exists, create one. The
		// integration test prefix makes it discoverable by the
		// startup/teardown sweep regardless of how the test exits.
		t.Log("no future-deadline datacall present; using a synthesized one for this test")
		name := fmt.Sprintf("%sfuture_%d", integrationTestPrefix, time.Now().UnixNano())
		err = conn.QueryRow(ctx, `
			INSERT INTO datacalls (datacall, datecreated, deadline)
			VALUES ($1, NOW(), NOW() + INTERVAL '7 days')
			RETURNING datacallid
		`, name).Scan(&futureDataCallID)
		require.NoError(t, err)
	}

	t.Run("NotesTooLong_ReturnsErr", func(t *testing.T) {
		bigNotes := strings.Repeat("x", 2001)
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       futureDataCallID,
			Notes:            &bigNotes,
		}
		_, err := s.Save(ctx)
		assert.ErrorIs(t, err, ErrNotesTooLong, "2001-char notes must trip the length guard")
	})

	t.Run("PastDeadline_NonAdmin_ReturnsErr", func(t *testing.T) {
		notes := "after deadline"
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       pastDataCallID,
			Notes:            &notes,
		}
		// No user in context = treated as non-admin per validate()
		_, err := s.Save(ctx)
		assert.ErrorIs(t, err, ErrPastDeadline,
			"past-deadline save without admin must trip deadline guard")
	})

	t.Run("PastDeadline_Admin_Allowed", func(t *testing.T) {
		notes := "after deadline but admin"
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       pastDataCallID,
			Notes:            &notes,
		}
		adminCtx := UserToContext(ctx, &User{
			UserID: "00000000-0000-4000-8000-000000000000",
			Role:   "ADMIN",
		})
		// Admin bypasses the deadline guard. The save itself may fail
		// for unrelated reasons (FK violation if system 1 / option 1
		// not seeded) — but the error must NOT be ErrPastDeadline.
		_, saveErr := s.Save(adminCtx)
		if saveErr != nil {
			assert.NotErrorIs(t, saveErr, ErrPastDeadline,
				"admin save past deadline must not be blocked by the deadline guard")
		}
		// Best-effort cleanup if it did succeed
		if s.ScoreID != 0 {
			_, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, s.ScoreID)
		}
	})
}

// TestCopyPreviousScoresIntegration verifies the rollover function
// carries forward scores from one datacall to the next. Creates two
// synthetic datacalls, attaches scores to the first, runs the copy,
// asserts the second now has the same set of (functionoptionid,
// fismasystemid) entries.
func TestCopyPreviousScoresIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close(ctx)

	// Insert two datacalls explicitly ordered so copyPreviousScores'
	// "latest-1 is previous" logic finds the right one. Each row gets
	// a unique nano-precision suffix so reruns don't collide on the
	// UNIQUE(datacall) constraint, and the integrationTestPrefix makes
	// them discoverable by the sweep no matter how the test exits.
	var prevDC, newDC int32
	prevTimestamp := time.Now().Add(-2 * time.Hour)
	newTimestamp := time.Now().Add(-1 * time.Hour)
	suffix := time.Now().UnixNano()

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%sprev_%d", integrationTestPrefix, suffix), prevTimestamp).Scan(&prevDC)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%snew_%d", integrationTestPrefix, suffix), newTimestamp).Scan(&newDC)
	require.NoError(t, err)

	// Pick a (fismasystemid, functionoptionid) that already references
	// a real function so the FK constraint is satisfied. Any active
	// system with at least one scored functionoption will do.
	var fismaSystemID, functionOptionID int32
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.functionoptionid
		FROM scores s LIMIT 1
	`).Scan(&fismaSystemID, &functionOptionID)
	require.NoError(t, err, "need at least one existing score row to derive a valid (system, functionoption) pair")

	// Insert a marker score under the prev datacall.
	notes := "integration test marker"
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, $4)
	`, fismaSystemID, functionOptionID, prevDC, notes)
	require.NoError(t, err)

	// Before copy: the new datacall has zero scores.
	var beforeCount int
	err = conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM scores WHERE datacallid = $1`, newDC,
	).Scan(&beforeCount)
	require.NoError(t, err)
	require.Equal(t, 0, beforeCount, "newDC starts with no scores")

	// Run the rollover. copyPreviousScores is unexported and accepts
	// the *latest* datacallid — it discovers the previous one via
	// findPreviousDataCall.
	copyPreviousScores(newDC)

	// After copy: the new datacall has at least the marker score.
	var afterCount int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM scores
		WHERE datacallid = $1
		  AND fismasystemid = $2
		  AND functionoptionid = $3
	`, newDC, fismaSystemID, functionOptionID).Scan(&afterCount)
	require.NoError(t, err)
	assert.Equal(t, 1, afterCount,
		"copyPreviousScores must carry the marker (system, functionoption) into the new datacall")
}

func boolPtr(b bool) *bool { return &b }

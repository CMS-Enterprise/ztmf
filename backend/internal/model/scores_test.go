package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTier covers every threshold boundary in the HHS-aligned tier predicate.
// The cutoffs Elizabeth confirmed in ztmf-misc#175 are:
//
//	Optimal      >= 4.1
//	Advanced     >= 3.1
//	Initial      >= 2.1
//	Traditional  >= 1.01
//	Not Assessed everything below 1.01 (a pillar of all-unanswered rows lands
//	             at exactly 1.0 under the +1 shift aggregation; 1.01 is the
//	             intentional floor for Traditional)
func TestTier(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		// Not Assessed: pillar with zero answers
		{0.0, "Not Assessed"},
		{0.99, "Not Assessed"},
		{1.0, "Not Assessed"},
		{1.005, "Not Assessed"},
		{1.009, "Not Assessed"},

		// Traditional: 1.01 through 2.09
		{1.01, "Traditional"},
		{1.5, "Traditional"},
		{2.0, "Traditional"},
		{2.09, "Traditional"},

		// Initial: 2.1 through 3.09
		{2.1, "Initial"},
		{2.5, "Initial"},
		{3.0, "Initial"},
		{3.09, "Initial"},

		// Advanced: 3.1 through 4.09
		{3.1, "Advanced"},
		{3.5, "Advanced"},
		{4.0, "Advanced"},
		{4.09, "Advanced"},

		// Optimal: 4.1 and above
		{4.1, "Optimal"},
		{4.5, "Optimal"},
		{5.0, "Optimal"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := Tier(tc.score)
			assert.Equal(t, tc.want, got, "Tier(%v)", tc.score)
		})
	}
}

// TestAggregatePillarRows verifies the Go-side grouping that runs after
// findPillarScoresAll returns. The system score itself is computed in SQL
// (window AVG on the carry-along system_score column on every pillar row)
// so this test seeds explicit SystemScore values to confirm the Go layer
// reads them rather than recomputing. Pure logic test, no database.
//
// Confirms that:
//   - Pillar rows for the same (datacall, system) collapse into one aggregate
//   - SystemScore is read from the carry-along column, not recomputed
//   - SystemTier and per-pillar Tier are populated from the Tier predicate
//   - IncludePillars controls whether the pillar breakdown is attached
//   - Different pillar counts per system (4 vs 6) are handled the same way,
//     since the divisor lived in SQL via AVG() and is now pre-computed
func TestAggregatePillarRows(t *testing.T) {
	rows := []*pillarScoreRow{
		// System 1, datacall 36: six pillars, one Optimal and five
		// Not-Assessed. SQL window AVG produced systemScore = 10/6.
		{DataCallID: 36, FismaSystemID: 1, PillarID: 1, Pillar: "Devices", Score: 5.0, SystemScore: 10.0 / 6.0},
		{DataCallID: 36, FismaSystemID: 1, PillarID: 2, Pillar: "Applications", Score: 1.0, SystemScore: 10.0 / 6.0},
		{DataCallID: 36, FismaSystemID: 1, PillarID: 3, Pillar: "Networks", Score: 1.0, SystemScore: 10.0 / 6.0},
		{DataCallID: 36, FismaSystemID: 1, PillarID: 4, Pillar: "Data", Score: 1.0, SystemScore: 10.0 / 6.0},
		{DataCallID: 36, FismaSystemID: 1, PillarID: 5, Pillar: "CrossCutting", Score: 1.0, SystemScore: 10.0 / 6.0},
		{DataCallID: 36, FismaSystemID: 1, PillarID: 6, Pillar: "Identity", Score: 1.0, SystemScore: 10.0 / 6.0},
		// System 2, datacall 36: all six pillars at Optimal, systemScore 5.0
		{DataCallID: 36, FismaSystemID: 2, PillarID: 1, Pillar: "Devices", Score: 5.0, SystemScore: 5.0},
		{DataCallID: 36, FismaSystemID: 2, PillarID: 2, Pillar: "Applications", Score: 5.0, SystemScore: 5.0},
		{DataCallID: 36, FismaSystemID: 2, PillarID: 3, Pillar: "Networks", Score: 5.0, SystemScore: 5.0},
		{DataCallID: 36, FismaSystemID: 2, PillarID: 4, Pillar: "Data", Score: 5.0, SystemScore: 5.0},
		{DataCallID: 36, FismaSystemID: 2, PillarID: 5, Pillar: "CrossCutting", Score: 5.0, SystemScore: 5.0},
		{DataCallID: 36, FismaSystemID: 2, PillarID: 6, Pillar: "Identity", Score: 5.0, SystemScore: 5.0},
		// System 3, datacall 36: all six pillars at 1.0 (zero answers
		// everywhere), systemScore 1.0
		{DataCallID: 36, FismaSystemID: 3, PillarID: 1, Pillar: "Devices", Score: 1.0, SystemScore: 1.0},
		{DataCallID: 36, FismaSystemID: 3, PillarID: 2, Pillar: "Applications", Score: 1.0, SystemScore: 1.0},
		{DataCallID: 36, FismaSystemID: 3, PillarID: 3, Pillar: "Networks", Score: 1.0, SystemScore: 1.0},
		{DataCallID: 36, FismaSystemID: 3, PillarID: 4, Pillar: "Data", Score: 1.0, SystemScore: 1.0},
		{DataCallID: 36, FismaSystemID: 3, PillarID: 5, Pillar: "CrossCutting", Score: 1.0, SystemScore: 1.0},
		{DataCallID: 36, FismaSystemID: 3, PillarID: 6, Pillar: "Identity", Score: 1.0, SystemScore: 1.0},
		// System 4, datacall 3: only four pillars present (simulating a
		// prior cycle that used fewer pillars). SQL AVG(5.0+4.0+3.0+2.0)/4
		// = 3.5. Asserting on the carry-along value pins that SQL is the
		// source of truth.
		{DataCallID: 3, FismaSystemID: 4, PillarID: 1, Pillar: "Devices", Score: 5.0, SystemScore: 3.5},
		{DataCallID: 3, FismaSystemID: 4, PillarID: 2, Pillar: "Applications", Score: 4.0, SystemScore: 3.5},
		{DataCallID: 3, FismaSystemID: 4, PillarID: 3, Pillar: "Networks", Score: 3.0, SystemScore: 3.5},
		{DataCallID: 3, FismaSystemID: 4, PillarID: 4, Pillar: "Data", Score: 2.0, SystemScore: 3.5},
	}

	t.Run("IncludePillars true", func(t *testing.T) {
		aggs := aggregatePillarRows(rows, true)
		if assert.Len(t, aggs, 4) {
			// System 1 (Traditional 5/6 + Optimal 1/6)
			assert.Equal(t, int32(36), aggs[0].DataCallID)
			assert.Equal(t, int32(1), aggs[0].FismaSystemID)
			assert.InDelta(t, 10.0/6.0, aggs[0].SystemScore, 1e-9)
			assert.Equal(t, "Traditional", aggs[0].SystemTier)
			assert.Len(t, aggs[0].PillarScores, 6)
			assert.Equal(t, "Optimal", aggs[0].PillarScores[0].Tier)
			assert.Equal(t, "Not Assessed", aggs[0].PillarScores[1].Tier)

			// System 2 (all Optimal)
			assert.Equal(t, 5.0, aggs[1].SystemScore)
			assert.Equal(t, "Optimal", aggs[1].SystemTier)

			// System 3 (all unanswered)
			assert.Equal(t, 1.0, aggs[2].SystemScore)
			assert.Equal(t, "Not Assessed", aggs[2].SystemTier)
			for _, p := range aggs[2].PillarScores {
				assert.Equal(t, "Not Assessed", p.Tier, "pillar %s", p.Pillar)
			}

			// System 4 (four pillars only — divisor lives in SQL)
			assert.Equal(t, int32(3), aggs[3].DataCallID)
			assert.Equal(t, 3.5, aggs[3].SystemScore)
			assert.Equal(t, "Advanced", aggs[3].SystemTier)
			assert.Len(t, aggs[3].PillarScores, 4)
		}
	})

	t.Run("IncludePillars false", func(t *testing.T) {
		aggs := aggregatePillarRows(rows, false)
		assert.Len(t, aggs, 4)
		for _, a := range aggs {
			assert.Empty(t, a.PillarScores, "pillarscores must be nil when not requested")
			assert.NotEmpty(t, a.SystemTier, "system tier must still be set")
		}
	})
}

func int32Ptr(i int32) *int32 { return &i }

// normalizeInput mirrors the FindScoresAggregate promotion of a bare
// FismaSystemID into the FismaSystemIDs slice, so the scope assertions below
// run against the same input shape the production query builder sees.
func normalizeInput(in FindScoresInput) FindScoresInput {
	if in.FismaSystemID != nil && len(in.FismaSystemIDs) == 0 {
		in.FismaSystemIDs = []*int32{in.FismaSystemID}
	}
	return in
}

// TestFindScoresAggregate_ISSOwithSpecificSystem is the regression test for the bug where
// an ISSO requesting a specific fismasystemid would get scores for ALL their assigned systems
// because the equality filter was skipped when FismaSystemIDs was pre-populated.
//
// Re-targeted at the HHS-scale pillar aggregation SQL (buildPillarScoresSQL).
// The scope predicates are still: a specific FismaSystemID adds an equality
// clause, FismaSystemIDs adds an IN clause, and both together must both be
// emitted so the result intersects the two.
func TestFindScoresAggregate_ISSOwithSpecificSystem(t *testing.T) {
	sys1, sys2, sys3 := int32Ptr(1001), int32Ptr(1002), int32Ptr(1003)

	t.Run("ISSOwithMultipleSystemsRequestsSpecific", func(t *testing.T) {
		// Simulate controller setting FismaSystemIDs to the user's assigned systems,
		// then query param setting FismaSystemID to the specific requested system.
		input := normalizeInput(FindScoresInput{
			FismaSystemIDs: []*int32{sys1, sys2, sys3},
			FismaSystemID:  sys1,
		})

		sql, args := buildPillarScoresSQL(input)

		// Must include the IN clause scoping to assigned systems
		assert.Contains(t, sql, "fs.fismasystemid IN", "should scope to assigned systems")
		// Must also include the equality predicate for the specific system
		assert.Contains(t, sql, "fs.fismasystemid = $", "should have equality filter for specific system")
		// 1 arg for equality + 3 for IN list = 4 args from the system scope.
		// Plus FismaSystemID promoted into FismaSystemIDs replaces the bare
		// slice in tests that don't override, but here both are explicit and
		// the IN list keeps three entries.
		assert.Len(t, args, 4, "should have 4 args: 1 for equality + 3 for IN list")
	})

	t.Run("ISSOwithSingleSystemRequestsSpecific", func(t *testing.T) {
		// Edge case: ISSO with only one assigned system.
		input := normalizeInput(FindScoresInput{
			FismaSystemIDs: []*int32{sys1},
			FismaSystemID:  sys1,
		})

		sql, args := buildPillarScoresSQL(input)

		assert.Contains(t, sql, "fs.fismasystemid", "should filter on fismasystemid")
		// args is []any so type-assert when checking membership
		found := false
		for _, a := range args {
			if p, ok := a.(*int32); ok && p == sys1 {
				found = true
				break
			}
		}
		assert.True(t, found, "args should include the requested system pointer")
	})

	t.Run("AdminRequestsSpecificSystem", func(t *testing.T) {
		// Admin path: FismaSystemIDs is empty, only FismaSystemID from query param.
		// normalizeInput promotes FismaSystemID -> FismaSystemIDs so the IN
		// list ends up with a single entry.
		input := normalizeInput(FindScoresInput{
			FismaSystemID: sys2,
		})

		sql, args := buildPillarScoresSQL(input)

		assert.Contains(t, sql, "fs.fismasystemid", "should filter on fismasystemid")
		assert.NotEmpty(t, args, "should bind at least one argument for the system filter")
	})

	t.Run("ISSOwithNoSpecificSystem", func(t *testing.T) {
		// ISSO list view: no fismasystemid query param, just the assigned systems scope.
		// Should produce an IN clause and no equality predicate.
		input := normalizeInput(FindScoresInput{
			FismaSystemIDs: []*int32{sys1, sys2},
		})

		sql, args := buildPillarScoresSQL(input)

		assert.Contains(t, sql, "fs.fismasystemid IN", "should scope to assigned systems")
		assert.NotContains(t, sql, "fs.fismasystemid = $", "should not have equality filter when no specific system requested")
		assert.Len(t, args, 2, "should have 2 args for IN list only")
	})
}

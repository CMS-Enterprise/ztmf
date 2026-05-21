package model

import (
	"testing"
	"time"

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
		// Not Assessed: the score rounds to less than 1.01.
		// (A pillar with zero answers lands at exactly 1.0 under the
		// +1 shift aggregation.)
		{0.0, "Not Assessed"},
		{0.99, "Not Assessed"},
		{1.0, "Not Assessed"},
		{1.004, "Not Assessed"},

		// Traditional: rounds to [1.01, 2.10).
		// 1.005 is intentionally not tested: the float64 representation
		// of "1.005" is 1.00499...8, just below the half-way point, so
		// math.Round produces 100 not 101. Real aggregations never land
		// at exactly 1.005 (the +1 shift over integer answers can't
		// produce it), and the only way to hit it is a hand-crafted
		// float literal.
		{1.009, "Traditional"},
		{1.01, "Traditional"},
		{1.5, "Traditional"},
		{2.0, "Traditional"},
		{2.094, "Traditional"},

		// Initial: rounds to [2.10, 3.10).
		// 2.095 rounds to 2.10 -> Initial. This is the displayed-value
		// boundary that the previous direct-float comparison got wrong.
		{2.095, "Initial"},
		{2.1, "Initial"},
		{2.5, "Initial"},
		{3.0, "Initial"},
		{3.094, "Initial"},

		// Advanced: rounds to [3.10, 4.10).
		// 3.095 rounds to 3.10 -> Advanced. The synthetic boundary
		// search found 642 distinct system averages that displayed as
		// "3.10" but tiered as Initial under the previous predicate
		// because they were a few ulps below the float64 representation
		// of 3.1; that mis-classification is now impossible.
		{3.095, "Advanced"},
		{3.1, "Advanced"},
		{3.5, "Advanced"},
		{4.0, "Advanced"},
		{4.094, "Advanced"},

		// Optimal: rounds to >= 4.10.
		{4.095, "Optimal"},
		{4.1, "Optimal"},
		{4.5, "Optimal"},
		{5.0, "Optimal"},

		// Pathological float inputs that would mis-classify under the
		// previous direct comparison: a value just below the literal
		// representation of 3.1 due to float arithmetic still rounds
		// to 3.10 on display, so it must classify as Advanced.
		{3.0999999999999996, "Advanced"},
		{4.0999999999999996, "Optimal"},
		{2.0999999999999996, "Initial"},
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

// TestScoreAuditInfo verifies the Auditable contract: AuditInfo returns the
// two pointers exactly as set on the struct. Pure accessor, no logic, but
// the contract is what generic consumers (exports, admin views) will rely
// on so a regression here would silently break any future Auditable user.
func TestScoreAuditInfo(t *testing.T) {
	t.Run("PopulatedRow", func(t *testing.T) {
		ts := time.Date(2026, 4, 14, 22, 12, 40, 0, time.UTC)
		ref := &AuditRef{
			UserID: "11111111-1111-1111-1111-111111111111",
			Name:   "Grand Moff Tarkin",
			Email:  "Grand.Moff@DeathStar.Empire",
			Role:   "ADMIN",
		}
		s := &Score{LastEditedAt: &ts, LastEditedBy: ref}

		gotAt, gotBy := s.AuditInfo()
		assert.Equal(t, &ts, gotAt, "AuditInfo must return the same time pointer")
		assert.Equal(t, ref, gotBy, "AuditInfo must return the same AuditRef pointer")
	})

	t.Run("UnseededRow", func(t *testing.T) {
		// A row inserted outside the event-tracking write path (seed data)
		// has no recorded edit. AuditInfo returns (nil, nil) and the
		// JSON encoder drops both fields via omitempty.
		s := &Score{}
		gotAt, gotBy := s.AuditInfo()
		assert.Nil(t, gotAt)
		assert.Nil(t, gotBy)
	})

	// Static interface conformance: if Score ever stops satisfying
	// Auditable, this file fails to compile. Cheaper signal than a runtime
	// assertion.
	var _ Auditable = (*Score)(nil)
}

// TestScoresEqualForUpdate covers the no-op detection used by Save to
// short-circuit a PUT that did not change any answer field. Without this
// guard a read-through user (clicking Next through a questionnaire
// without editing) would overwrite the prior cycle's editor in the
// audit trail. The rule we pin: equal iff all answer fields match,
// with nil and empty-string notes treated as the same value because the
// FE may submit either for an unanswered notes box.
func TestScoresEqualForUpdate(t *testing.T) {
	base := func() *Score {
		notes := "the same"
		return &Score{
			ScoreID:          42,
			FismaSystemID:    1001,
			DataCallID:       3,
			FunctionOptionID: 1,
			Notes:            &notes,
		}
	}

	t.Run("Identical", func(t *testing.T) {
		assert.True(t, scoresEqualForUpdate(base(), base()))
	})

	t.Run("DifferentNotes", func(t *testing.T) {
		incoming := base()
		other := "different"
		incoming.Notes = &other
		assert.False(t, scoresEqualForUpdate(base(), incoming))
	})

	t.Run("DifferentFunctionOption", func(t *testing.T) {
		incoming := base()
		incoming.FunctionOptionID = 99
		assert.False(t, scoresEqualForUpdate(base(), incoming))
	})

	t.Run("DifferentFismaSystem", func(t *testing.T) {
		incoming := base()
		incoming.FismaSystemID = 1002
		assert.False(t, scoresEqualForUpdate(base(), incoming))
	})

	t.Run("DifferentDataCall", func(t *testing.T) {
		incoming := base()
		incoming.DataCallID = 4
		assert.False(t, scoresEqualForUpdate(base(), incoming))
	})

	t.Run("NilNotesEqualsEmptyString", func(t *testing.T) {
		current := base()
		current.Notes = nil
		incoming := base()
		empty := ""
		incoming.Notes = &empty
		assert.True(t, scoresEqualForUpdate(current, incoming),
			"nil and empty-string notes must compare equal because the FE may submit either")
	})

	t.Run("BothNotesNil", func(t *testing.T) {
		current := base()
		current.Notes = nil
		incoming := base()
		incoming.Notes = nil
		assert.True(t, scoresEqualForUpdate(current, incoming))
	})

	t.Run("NilInputsReturnFalse", func(t *testing.T) {
		assert.False(t, scoresEqualForUpdate(nil, base()))
		assert.False(t, scoresEqualForUpdate(base(), nil))
	})

	// Pinned contract: whitespace-only notes do NOT normalize to empty.
	// The FE trims before submitting today, so reaching this comparison
	// with " " on one side means a caller is deliberately sending
	// whitespace and the stored value should reflect that. If a future
	// change wants to add whitespace tolerance, this test will fail and
	// force an intentional update to the rule rather than silent drift.
	t.Run("WhitespaceNotesNotEqualEmpty", func(t *testing.T) {
		current := base()
		empty := ""
		current.Notes = &empty
		incoming := base()
		space := " "
		incoming.Notes = &space
		assert.False(t, scoresEqualForUpdate(current, incoming),
			"whitespace-only notes must be treated as a real value distinct from empty until the FE/BE agree to trim on both sides")
	})
}

// TestDerefString covers the nil-safe string deref used when building an
// AuditRef from nullable scan targets. The LEFT JOIN can return all-nulls
// for the editor block when a score row has no event; the helper turns
// those into zero values without panicking.
func TestDerefString(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.Equal(t, "", derefString(nil))
	})
	t.Run("Value", func(t *testing.T) {
		v := "ADMIN"
		assert.Equal(t, "ADMIN", derefString(&v))
	})
	t.Run("EmptyStringPointer", func(t *testing.T) {
		v := ""
		assert.Equal(t, "", derefString(&v),
			"pointer to empty string is distinct from nil and must round-trip")
	})
}

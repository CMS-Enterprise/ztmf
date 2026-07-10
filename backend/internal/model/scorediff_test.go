package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindScoreDiffInputValidate pins the request preconditions: both data call
// IDs are required, and the two cycles must differ (a diff of a cycle against
// itself is always empty and almost certainly a client mistake). The error is
// an *InvalidInputError so the controller surfaces it as a 400 with the
// offending fields, matching the rest of the API.
func TestFindScoreDiffInputValidate(t *testing.T) {
	t.Run("BothPresentAndDistinct", func(t *testing.T) {
		in := FindScoreDiffInput{FromDataCallID: int32Ptr(3), ToDataCallID: int32Ptr(4)}
		assert.NoError(t, in.validate())
	})

	t.Run("MissingFrom", func(t *testing.T) {
		in := FindScoreDiffInput{ToDataCallID: int32Ptr(4)}
		err := in.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "from")
		}
	})

	t.Run("MissingTo", func(t *testing.T) {
		in := FindScoreDiffInput{FromDataCallID: int32Ptr(3)}
		err := in.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "to")
		}
	})

	t.Run("MissingBoth", func(t *testing.T) {
		err := FindScoreDiffInput{}.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "from")
			assert.Contains(t, iie.Data(), "to")
		}
	})

	t.Run("SameCycle", func(t *testing.T) {
		in := FindScoreDiffInput{FromDataCallID: int32Ptr(5), ToDataCallID: int32Ptr(5)}
		err := in.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "to")
		}
	})
}

// TestBuildScoreDiffSQL_Shape verifies the structural invariants of the query:
// it diffs two cycles via a FULL OUTER JOIN (so a one-sided answer surfaces),
// drops unchanged rows with the IS DISTINCT FROM pair, treats nil/empty notes
// as equal, and attributes the change to the later (To) write through the
// events lateral. The two cycle CTEs each bind their own datacallid, so the
// from/to ids appear once each in arg order.
func TestBuildScoreDiffSQL_Shape(t *testing.T) {
	in := FindScoreDiffInput{FromDataCallID: int32Ptr(3), ToDataCallID: int32Ptr(4)}
	sql, args := buildScoreDiffSQL(in)

	assert.Contains(t, sql, "FULL OUTER JOIN to_scores", "must outer-join the two cycles so one-sided answers surface")
	assert.Contains(t, sql, "f.functionoptionid IS DISTINCT FROM t.functionoptionid", "must drop rows with an unchanged option")
	assert.Contains(t, sql, `regexp_replace(COALESCE(f.notes, ''), '\s+', ' ', 'g')`, "notes must be whitespace-normalized before comparison (nil/empty equal, spacing-only differences ignored)")
	assert.Contains(t, sql, "resource = 'public.scores'", "attribution lateral must read score events")
	assert.Contains(t, sql, "(payload->>'scoreid')::int = t.scoreid", "attribution must key on the later (To) write")

	// Two cycle filters, no scope: from=3 then to=4, in that order.
	if assert.Len(t, args, 2, "expected one datacallid arg per cycle") {
		assert.Equal(t, int32(3), args[0], "from cycle binds first")
		assert.Equal(t, int32(4), args[1], "to cycle binds second")
	}
}

// TestBuildScoreDiffSQL_FismaSystemScope verifies that a single-system request
// adds the equality predicate to BOTH cycle CTEs (the diff is meaningless if
// only one side is narrowed), binding the system id once per cycle.
func TestBuildScoreDiffSQL_FismaSystemScope(t *testing.T) {
	in := FindScoreDiffInput{
		FromDataCallID: int32Ptr(3),
		ToDataCallID:   int32Ptr(4),
		FismaSystemID:  int32Ptr(1001),
	}
	sql, args := buildScoreDiffSQL(in)

	assert.Equal(t, 2, strings.Count(sql, "s.fismasystemid = $"), "system filter must apply to both cycle CTEs")
	// from: datacall, system ; to: datacall, system
	assert.Equal(t, []any{int32(3), int32(1001), int32(4), int32(1001)}, args)
}

// TestBuildScoreDiffSQL_UserScope verifies the ISSO/ISSM path: each cycle CTE
// inner-joins users_fismasystems on the requesting user so the diff only spans
// their assigned systems, binding the user id once per cycle ahead of that
// cycle's datacallid.
func TestBuildScoreDiffSQL_UserScope(t *testing.T) {
	uid := "11111111-1111-1111-1111-111111111111"
	in := FindScoreDiffInput{
		FromDataCallID: int32Ptr(3),
		ToDataCallID:   int32Ptr(4),
		UserID:         &uid,
	}
	sql, args := buildScoreDiffSQL(in)

	assert.Equal(t, 2, strings.Count(sql, "INNER JOIN users_fismasystems"), "user scope must join both cycle CTEs")
	// from: userid, datacall ; to: userid, datacall
	assert.Equal(t, []any{uid, int32(3), uid, int32(4)}, args)
}

// TestBuildScoreDiffSQL_OpDivScope mirrors the OpDiv read-scope contract from
// the aggregate query: granted OpDivs emit a subquery predicate on each cycle
// and bind the slice; a restricted-but-empty grant set fails closed with FALSE
// so a scoped admin with no grants matches nothing rather than everything.
func TestBuildScoreDiffSQL_OpDivScope(t *testing.T) {
	t.Run("ScopedToGrantedOpDivs", func(t *testing.T) {
		in := FindScoreDiffInput{
			FromDataCallID:     int32Ptr(3),
			ToDataCallID:       int32Ptr(4),
			RestrictToOpDivIDs: true,
			OpDivIDs:           []int32{7, 9},
		}
		sql, args := buildScoreDiffSQL(in)

		assert.Equal(t, 2, strings.Count(sql, "opdiv_id = ANY($"), "OpDiv scope must apply to both cycle CTEs")
		assert.NotContains(t, sql, "FALSE", "non-empty grants should not fail closed")
		// The slice is bound twice (once per cycle).
		slices := 0
		for _, a := range args {
			if ids, ok := a.([]int32); ok && len(ids) == 2 && ids[0] == 7 && ids[1] == 9 {
				slices++
			}
		}
		assert.Equal(t, 2, slices, "the OpDiv id slice should be bound once per cycle")
	})

	t.Run("RestrictedWithNoGrantsFailsClosed", func(t *testing.T) {
		in := FindScoreDiffInput{
			FromDataCallID:     int32Ptr(3),
			ToDataCallID:       int32Ptr(4),
			RestrictToOpDivIDs: true,
		}
		sql, _ := buildScoreDiffSQL(in)

		assert.Equal(t, 2, strings.Count(sql, "FALSE"), "a scoped admin with no grants must match nothing in both cycles")
		assert.NotContains(t, sql, "opdiv_id = ANY($", "should not bind an OpDiv predicate when there are no grants")
	})

	t.Run("UnscopedEmitsNoOpDivPredicate", func(t *testing.T) {
		in := FindScoreDiffInput{FromDataCallID: int32Ptr(3), ToDataCallID: int32Ptr(4)}
		sql, _ := buildScoreDiffSQL(in)

		assert.NotContains(t, sql, "opdiv_id = ANY($", "unscoped admins get no OpDiv filter")
		assert.NotContains(t, sql, "FALSE", "unscoped admins do not fail closed")
	})
}

// TestDerefInt32 covers the nil-safe int32 deref used when building a
// ScoreDiffSide from nullable scan targets on the unmatched side of the FULL
// OUTER JOIN.
func TestDerefInt32(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.Equal(t, int32(0), derefInt32(nil))
	})
	t.Run("Value", func(t *testing.T) {
		v := int32(7)
		assert.Equal(t, int32(7), derefInt32(&v))
	})
}

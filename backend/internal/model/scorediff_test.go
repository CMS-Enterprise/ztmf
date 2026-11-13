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
// events lateral. Each cycle CTE binds its own datacallid, then the other
// side's for the reduced pillar scope, so the ids appear twice in arg order.
func TestBuildScoreDiffSQL_Shape(t *testing.T) {
	in := FindScoreDiffInput{FromDataCallID: int32Ptr(3), ToDataCallID: int32Ptr(4)}
	sql, args := buildScoreDiffSQL(in)

	assert.Contains(t, sql, "FULL OUTER JOIN to_scores", "must outer-join the two cycles so one-sided answers surface")
	assert.Contains(t, sql, "f.functionoptionid IS DISTINCT FROM t.functionoptionid", "must drop rows with an unchanged option")
	assert.Contains(t, sql, `regexp_replace(COALESCE(f.notes, ''), '\s+', ' ', 'g')`, "notes must be whitespace-normalized before comparison (nil/empty equal, spacing-only differences ignored)")
	assert.Contains(t, sql, "resource = 'public.scores'", "attribution lateral must read score events")
	assert.Contains(t, sql, "(payload->>'scoreid')::int = t.scoreid", "attribution must key on the later (To) write")
	for _, dc := range []string{"$1", "$3"} {
		assert.Contains(t, sql, "s.datacallid = "+dc+" AND "+questionnaireVersionSQL("rf.versionid", dc),
			"each cycle reads only its own pinned version, on its own datacallid placeholder")
	}
	// ztmf#645: each side applies the reduced pillar scope for both calls.
	assert.Contains(t, sql, reducedPillarScopeAnySQL("sdce.scoring_key", "sp.pillar", "$1", "$2"))
	assert.Contains(t, sql, reducedPillarScopeAnySQL("sdce.scoring_key", "sp.pillar", "$3", "$4"))

	// from=3 (own, then other 4), to=4 (own, then other 3).
	assert.Equal(t, []any{int32(3), int32(4), int32(4), int32(3)}, args)
}

// The diff's scope must not drop a row through a join: an unmapped or
// NULL-key environment, or a function with no question, reaches NOT EXISTS
// as NULL and keeps the row (ztmf#645). The lookups are LEFT JOINs on the
// cycle CTE; an inner join there would drop the answer.
func TestBuildScoreDiffSQL_ReducedScopeIsNullSafe(t *testing.T) {
	in := FindScoreDiffInput{FromDataCallID: int32Ptr(3), ToDataCallID: int32Ptr(4)}
	sql, _ := buildScoreDiffSQL(in)
	assert.Equal(t, 2, strings.Count(sql, "NOT EXISTS ("), "one predicate per side, covering both calls")
	assert.Equal(t, 2, strings.Count(sql, scoreDiffScopeJoins), "both cycle CTEs resolve the scope through the same joins")

	// Each cycle CTE's own FROM: every join to a lookup table must be LEFT.
	for _, cte := range strings.Split(sql, "FROM scores s")[1:] {
		from := cte[:strings.Index(cte, "WHERE s.datacallid")]
		for _, table := range []string{"fismasystems", "datacenterenvironments", "questions", "pillars"} {
			lefts := strings.Count(from, "LEFT JOIN "+table+" ")
			all := strings.Count(from, "JOIN "+table+" ")
			assert.Equal(t, 1, lefts, "%s is joined once, LEFT", table)
			assert.Equal(t, lefts, all, "an inner join to %s on the cycle CTE could drop an answer", table)
		}
	}
}

// TestBuildScoreDiffSQL_AllFiltersTogether pins placeholder order when every
// filter applies: per side the user, its own call, the other call, the
// system, then the OpDiv grants.
func TestBuildScoreDiffSQL_AllFiltersTogether(t *testing.T) {
	uid := "11111111-1111-1111-1111-111111111111"
	in := FindScoreDiffInput{
		FromDataCallID: int32Ptr(3),
		ToDataCallID:   int32Ptr(4),
		FismaSystemID:  int32Ptr(1001),
		UserID:         &uid,
		OpDivScope:     OpDivScope{RestrictToOpDivIDs: true, OpDivIDs: []int32{7, 9}},
	}
	sql, args := buildScoreDiffSQL(in)

	grants := []int32{7, 9}
	assert.Equal(t, []any{
		uid, int32(3), int32(4), int32(1001), grants,
		uid, int32(4), int32(3), int32(1001), grants,
	}, args)

	for _, side := range []struct{ user, own, other, system, opdiv string }{
		{"$1", "$2", "$3", "$4", "$5"},
		{"$6", "$7", "$8", "$9", "$10"},
	} {
		assert.Contains(t, sql, "ufs.userid = "+side.user)
		assert.Contains(t, sql, "s.datacallid = "+side.own+" AND "+questionnaireVersionSQL("rf.versionid", side.own))
		assert.Contains(t, sql, "tdc.datacallid IN ("+side.own+", "+side.other+")")
		assert.Contains(t, sql, "s.fismasystemid = "+side.system)
		assert.Contains(t, sql, "opdiv_id = ANY("+side.opdiv+")")
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
	// from: datacall, other datacall, system ; to: the same
	assert.Equal(t, []any{int32(3), int32(4), int32(1001), int32(4), int32(3), int32(1001)}, args)
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
	for _, dc := range []string{"$2", "$5"} {
		assert.Contains(t, sql, questionnaireVersionSQL("rf.versionid", dc), "the version pin follows the datacallid, not the user id")
	}
	// from: userid, datacall, other datacall ; to: the same
	assert.Equal(t, []any{uid, int32(3), int32(4), uid, int32(4), int32(3)}, args)
}

// TestBuildScoreDiffSQL_OpDivScope mirrors the OpDiv read-scope contract from
// the aggregate query: granted OpDivs emit a subquery predicate on each cycle
// and bind the slice; a restricted-but-empty grant set fails closed with FALSE
// so a scoped admin with no grants matches nothing rather than everything.
func TestBuildScoreDiffSQL_OpDivScope(t *testing.T) {
	t.Run("ScopedToGrantedOpDivs", func(t *testing.T) {
		in := FindScoreDiffInput{
			FromDataCallID: int32Ptr(3),
			ToDataCallID:   int32Ptr(4),
			OpDivScope:     OpDivScope{RestrictToOpDivIDs: true, OpDivIDs: []int32{7, 9}},
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
			FromDataCallID: int32Ptr(3),
			ToDataCallID:   int32Ptr(4),
			OpDivScope:     OpDivScope{RestrictToOpDivIDs: true},
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

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindScoreProgressInputValidate pins the request precondition: the data
// call id is required (progress is meaningless without a cycle to measure).
// The error is an *InvalidInputError so the controller surfaces it as a 400
// with the offending field, matching the rest of the API.
func TestFindScoreProgressInputValidate(t *testing.T) {
	t.Run("DataCallPresent", func(t *testing.T) {
		in := FindScoreProgressInput{DataCallID: int32Ptr(4)}
		assert.NoError(t, in.validate())
	})

	t.Run("MissingDataCall", func(t *testing.T) {
		err := FindScoreProgressInput{}.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "datacallid")
		}
	})
}

// TestBuildScoreProgressSQL_Shape verifies the structural invariants of the
// query:
//
//   - scope is applied once in a scoped_systems anchor CTE that both count
//     halves read from, so updated never computes for out-of-scope systems;
//   - both expected and updated resolve the applicable-function set the same
//     way FindQuestionsByFismaSystem does (functions + questions + pillars +
//     the datacenterenvironments vocabulary), so orphan functions are excluded
//     and the two halves draw from the same set - guaranteeing updated cannot
//     exceed expected;
//   - updated additionally requires the answered function to still be
//     applicable to the system's current environment (the dce join keyed on
//     both the system env and the function's scoring key), which is what stops
//     a carried-over answer to a now-inapplicable function from inflating the
//     numerator past 100%;
//   - the updated CTE keys on edit events via an INNER lateral, excluding
//     pre-populated rows (copyPreviousScores records no events);
//   - both halves LEFT JOIN back onto scoped_systems so a zero-activity or
//     unmapped-environment system still returns a row (0 of N).
func TestBuildScoreProgressSQL_Shape(t *testing.T) {
	in := FindScoreProgressInput{DataCallID: int32Ptr(4)}
	sql, args := buildScoreProgressSQL(in)

	assert.Contains(t, sql, "WITH scoped_systems AS", "scope is applied once in a shared anchor CTE")
	assert.Contains(t, sql, "fs.decommissioned = FALSE", "decommissioned systems do not participate in data calls and must not appear")
	// Both halves resolve applicability through the same canonical join chain.
	assert.Contains(t, sql, "dce.datacenterenvironment = ss.datacenterenvironment", "environment maps into the scoring vocabulary")
	assert.Contains(t, sql, "f.datacenterenvironment = dce.scoring_key", "functions match on the scoring key")
	assert.Contains(t, sql, "INNER JOIN questions q ON q.questionid = f.questionid", "orphan functions (no question) must be excluded, matching the questionnaire")
	assert.Contains(t, sql, "INNER JOIN pillars p ON p.pillarid = q.pillarid", "applicability mirrors FindQuestionsByFismaSystem")
	// updated re-checks applicability against the system's CURRENT environment.
	assert.Contains(t, sql, "dce.scoring_key = f.datacenterenvironment", "an answered function must still be applicable to the system's current environment")
	assert.Equal(t, 2, strings.Count(sql, "COUNT(DISTINCT f.functionid)"), "both halves count distinct applicable functions from the same set")
	assert.Contains(t, sql, "INNER JOIN LATERAL", "updated count must require an edit event so pre-populated rows drop out")
	assert.Contains(t, sql, "resource = 'public.scores'", "the lateral must read score events")
	assert.Contains(t, sql, "LEFT JOIN expected", "unmapped-environment systems must still return a row")
	assert.Contains(t, sql, "LEFT JOIN updated", "zero-activity systems must still return a row")
	assert.Contains(t, sql, "COALESCE(u.questionsupdated, 0)", "zero-activity systems report 0, not NULL")
	assert.Contains(t, sql, "COALESCE(ex.questionsexpected, 0)", "unmapped-environment systems report 0 expected, not NULL")

	// No scope filters: the single arg is the data call id.
	assert.Equal(t, []any{int32(4)}, args)
}

// TestBuildScoreProgressSQL_FismaSystemScope verifies a single-system request
// narrows the scoped_systems anchor (so both count halves only touch that
// system's rows) and binds the system id before the data call id.
func TestBuildScoreProgressSQL_FismaSystemScope(t *testing.T) {
	in := FindScoreProgressInput{
		DataCallID:    int32Ptr(4),
		FismaSystemID: int32Ptr(1001),
	}
	sql, args := buildScoreProgressSQL(in)

	assert.Contains(t, sql, "fs.fismasystemid = $1")
	assert.Equal(t, []any{int32(1001), int32(4)}, args)
}

// TestBuildScoreProgressSQL_UserScope verifies the ISSO/ISSM path: the
// scoped_systems anchor is restricted to the requesting user's assigned
// systems via a users_fismasystems subquery.
func TestBuildScoreProgressSQL_UserScope(t *testing.T) {
	uid := "11111111-1111-1111-1111-111111111111"
	in := FindScoreProgressInput{
		DataCallID: int32Ptr(4),
		UserID:     &uid,
	}
	sql, args := buildScoreProgressSQL(in)

	assert.Contains(t, sql, "SELECT fismasystemid FROM users_fismasystems WHERE userid = $1")
	assert.Equal(t, []any{uid, int32(4)}, args)
}

// TestBuildScoreProgressSQL_OpDivScope mirrors the OpDiv read-scope contract
// from FindScores: granted OpDivs emit an ANY predicate on the systems table;
// a restricted-but-empty grant set fails closed with FALSE so a scoped admin
// with no grants matches nothing rather than everything.
func TestBuildScoreProgressSQL_OpDivScope(t *testing.T) {
	t.Run("ScopedToGrantedOpDivs", func(t *testing.T) {
		in := FindScoreProgressInput{
			DataCallID:         int32Ptr(4),
			RestrictToOpDivIDs: true,
			OpDivIDs:           []int32{7, 9},
		}
		sql, args := buildScoreProgressSQL(in)

		assert.Contains(t, sql, "fs.opdiv_id = ANY($1)")
		assert.NotContains(t, sql, "AND FALSE", "non-empty grants should not fail closed")
		assert.Equal(t, []any{[]int32{7, 9}, int32(4)}, args)
	})

	t.Run("RestrictedWithNoGrantsFailsClosed", func(t *testing.T) {
		in := FindScoreProgressInput{
			DataCallID:         int32Ptr(4),
			RestrictToOpDivIDs: true,
		}
		sql, args := buildScoreProgressSQL(in)

		assert.True(t, strings.Contains(sql, "AND FALSE"), "a scoped admin with no grants must match nothing")
		assert.NotContains(t, sql, "opdiv_id = ANY($", "should not bind an OpDiv predicate when there are no grants")
		assert.Equal(t, []any{int32(4)}, args)
	})
}

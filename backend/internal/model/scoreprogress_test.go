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
//   - expected counts flow through the datacenterenvironments scoring
//     vocabulary (same indirection the questionnaire uses), with LEFT joins
//     so an unmapped environment still yields a zero-count row;
//   - the updated CTE keys on edit events via an INNER lateral, which is what
//     excludes pre-populated rows (copyPreviousScores records no events), the
//     core "has a row but was not updated" distinction of ztmf#299;
//   - the outer join is LEFT from expected so zero-activity systems still
//     return a row (that row is the entire point of the feature).
func TestBuildScoreProgressSQL_Shape(t *testing.T) {
	in := FindScoreProgressInput{DataCallID: int32Ptr(4)}
	sql, args := buildScoreProgressSQL(in)

	assert.Contains(t, sql, "fs.decommissioned = FALSE", "decommissioned systems do not participate in data calls and must not appear")
	assert.Contains(t, sql, "LEFT JOIN datacenterenvironments dce", "expected count must resolve environments through the scoring vocabulary")
	assert.Contains(t, sql, "dce.datacenterenvironment = fs.datacenterenvironment", "system's raw environment maps into the vocabulary")
	assert.Contains(t, sql, "f.datacenterenvironment = dce.scoring_key", "functions match on the scoring key")
	assert.Contains(t, sql, "INNER JOIN LATERAL", "updated count must require an edit event so pre-populated rows drop out")
	assert.Contains(t, sql, "resource = 'public.scores'", "the lateral must read score events")
	assert.Contains(t, sql, "COUNT(DISTINCT fo.functionid)", "updated counts distinct functions, not raw rows")
	assert.Contains(t, sql, "LEFT JOIN updated", "zero-activity systems must still return a row")
	assert.Contains(t, sql, "COALESCE(u.questionsupdated, 0)", "zero-activity systems report 0, not NULL")

	// No scope filters: the single arg is the data call id.
	assert.Equal(t, []any{int32(4)}, args)
}

// TestBuildScoreProgressSQL_FismaSystemScope verifies a single-system request
// narrows the expected CTE (which anchors the result set) and binds the system
// id before the data call id.
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
// expected CTE is restricted to the requesting user's assigned systems via a
// users_fismasystems subquery.
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

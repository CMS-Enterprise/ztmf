package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindTimeSpentInputValidate pins the precondition: a data call id is
// required (time spent is meaningless without a cycle to measure). The error is
// an *InvalidInputError so the controller surfaces it as a 400.
func TestFindTimeSpentInputValidate(t *testing.T) {
	t.Run("DataCallPresent", func(t *testing.T) {
		in := FindTimeSpentInput{DataCallID: int32Ptr(4)}
		assert.NoError(t, in.validate())
	})

	t.Run("MissingDataCall", func(t *testing.T) {
		err := FindTimeSpentInput{}.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "datacallid")
		}
	})
}

// TestIdleCapSeconds pins the idle cap at 20 minutes (issue #368 decision): a
// single view/gap can accrue at most 1200 seconds.
func TestIdleCapSeconds(t *testing.T) {
	assert.Equal(t, 20*60, idleCapSeconds)
	assert.Equal(t, 1200, idleCapSeconds)
}

// TestBuildTimeSpentSQL_MeasuredShape verifies the structural invariants of the
// measured (view-based) dwell query: scope is applied once in a shared anchor
// CTE; the stream windows over BOTH view pings and score saves so a view's next
// event bounds its dwell; only 'viewed' rows accrue time; each view's readonly
// flag splits the dwell into editor vs viewer buckets; and every dwell is
// clamped at the compile-time idle cap so an abandoned view cannot balloon a
// total.
func TestBuildTimeSpentSQL_MeasuredShape(t *testing.T) {
	in := FindTimeSpentInput{DataCallID: int32Ptr(4)}
	sql, args := buildTimeSpentSQL(in)

	assert.Contains(t, sql, "WITH scoped_systems AS", "scope is applied once in a shared anchor CTE")
	assert.Contains(t, sql, "resource IN ('questionnaire', 'public.scores')", "the stream spans view pings and saves so a save can bound a view's dwell")
	assert.Contains(t, sql, "LEAD(e.createdat) OVER", "each view's next event is found via LEAD over the ordered stream")
	assert.Contains(t, sql, "PARTITION BY e.userid, (e.payload->>'fismasystemid')::int", "the timeline is windowed per user per system")
	assert.Contains(t, sql, "WHERE action = 'viewed' AND next_at IS NOT NULL", "only views accrue time, and only when a following event exists")
	assert.Contains(t, sql, fmt.Sprintf("INTERVAL '%d seconds'", idleCapSeconds), "dwell is clamped at the idle cap")
	assert.Contains(t, sql, "LEAST(next_at - createdat,", "the clamp is a LEAST against the idle cap")
	assert.Contains(t, sql, "(e.payload->>'readonly')::boolean", "the view's readonly flag drives the editor/viewer split")
	assert.Contains(t, sql, "FILTER (WHERE NOT d.readonly)", "editor seconds are the non-readonly views")
	assert.Contains(t, sql, "FILTER (WHERE d.readonly)", "viewer seconds are the readonly views")
	assert.Contains(t, sql, "GROUP BY d.fismasystemid, d.userid, u.fullname, u.email, u.role, d.questionid", "rows are returned at the (system, user, question) grain")

	// No scope filters: the single arg is the data call id.
	assert.Equal(t, []any{int32(4)}, args)
}

// TestBuildTimeSpentSQL_FismaSystemScope verifies a single-system request
// narrows the scoped_systems anchor and binds the system id before the data
// call id.
func TestBuildTimeSpentSQL_FismaSystemScope(t *testing.T) {
	in := FindTimeSpentInput{
		DataCallID:    int32Ptr(4),
		FismaSystemID: int32Ptr(1001),
	}
	sql, args := buildTimeSpentSQL(in)

	assert.Contains(t, sql, "fs.fismasystemid = $1")
	assert.Equal(t, []any{int32(1001), int32(4)}, args)
}

// TestBuildTimeSpentSQL_UserScope verifies the ISSO/ISSM path: the
// scoped_systems anchor is restricted to the requesting user's assigned systems.
func TestBuildTimeSpentSQL_UserScope(t *testing.T) {
	uid := "11111111-1111-1111-1111-111111111111"
	in := FindTimeSpentInput{
		DataCallID: int32Ptr(4),
		UserID:     &uid,
	}
	sql, args := buildTimeSpentSQL(in)

	assert.Contains(t, sql, "SELECT fismasystemid FROM users_fismasystems WHERE userid = $1")
	assert.Equal(t, []any{uid, int32(4)}, args)
}

// TestBuildTimeSpentSQL_OpDivScope mirrors the OpDiv read-scope contract: a
// granted set emits an ANY predicate; a restricted-but-empty grant set fails
// closed with FALSE so a scoped admin with no grants matches nothing.
func TestBuildTimeSpentSQL_OpDivScope(t *testing.T) {
	t.Run("ScopedToGrantedOpDivs", func(t *testing.T) {
		in := FindTimeSpentInput{
			DataCallID:         int32Ptr(4),
			RestrictToOpDivIDs: true,
			OpDivIDs:           []int32{7, 9},
		}
		sql, args := buildTimeSpentSQL(in)

		assert.Contains(t, sql, "fs.opdiv_id = ANY($1)")
		assert.Equal(t, []any{[]int32{7, 9}, int32(4)}, args)
	})

	t.Run("RestrictedWithNoGrantsFailsClosed", func(t *testing.T) {
		in := FindTimeSpentInput{
			DataCallID:         int32Ptr(4),
			RestrictToOpDivIDs: true,
		}
		sql, args := buildTimeSpentSQL(in)

		assert.Contains(t, sql, "FALSE", "a scoped admin with no grants must match nothing")
		assert.Equal(t, []any{int32(4)}, args)
	})
}

// TestRollupTimeSpent verifies the (system, user, question) grain is folded
// into per-system totals, a per-person breakdown split into editor/viewer time,
// the average-per-question, and the per-question breakdown, independent of the
// database.
func TestRollupTimeSpent(t *testing.T) {
	rows := []*timeSpentRow{
		// System 1: user A on q10 (60s editor) and q11 (30s viewer); user B on
		// q10 (20s editor).
		{FismaSystemID: 1, UserID: "A", Name: "Alice", Email: "a@x", Role: "ISSO", QuestionID: 10, EditorSeconds: 60},
		{FismaSystemID: 1, UserID: "A", Name: "Alice", Email: "a@x", Role: "ISSO", QuestionID: 11, ViewerSeconds: 30},
		{FismaSystemID: 1, UserID: "B", Name: "Bob", Email: "b@x", Role: "ISSM", QuestionID: 10, EditorSeconds: 20},
		// System 2: user A on q20 (15s editor).
		{FismaSystemID: 2, UserID: "A", Name: "Alice", Email: "a@x", Role: "ISSO", QuestionID: 20, EditorSeconds: 15},
	}

	out := rollupTimeSpent(rows)
	assert.Len(t, out, 2, "one entry per system, first-seen order")

	sys1 := out[0]
	assert.Equal(t, int32(1), sys1.FismaSystemID)
	assert.Equal(t, float64(110), sys1.TotalSeconds, "60+30+20")
	assert.Equal(t, float64(80), sys1.EditorSeconds, "60 (A/q10) + 20 (B/q10)")
	assert.Equal(t, float64(30), sys1.ViewerSeconds, "30 (A/q11)")
	assert.Equal(t, int32(2), sys1.QuestionsMeasured, "distinct questions q10 and q11")
	assert.Equal(t, float64(55), sys1.AverageSecondsPerQuestion, "110 total / 2 distinct questions")
	assert.Len(t, sys1.PerPerson, 2)

	// Per-person breakdown, in first-seen order (A then B), with editor/viewer split.
	assert.Equal(t, "A", sys1.PerPerson[0].UserID)
	assert.Equal(t, "Alice", sys1.PerPerson[0].Name)
	assert.Equal(t, float64(90), sys1.PerPerson[0].TotalSeconds, "Alice: 60+30")
	assert.Equal(t, float64(60), sys1.PerPerson[0].EditorSeconds)
	assert.Equal(t, float64(30), sys1.PerPerson[0].ViewerSeconds)
	assert.Equal(t, int32(2), sys1.PerPerson[0].QuestionsMeasured)
	assert.Equal(t, "B", sys1.PerPerson[1].UserID)
	assert.Equal(t, float64(20), sys1.PerPerson[1].TotalSeconds)
	assert.Equal(t, float64(20), sys1.PerPerson[1].EditorSeconds)
	assert.Equal(t, float64(0), sys1.PerPerson[1].ViewerSeconds)
	assert.Equal(t, int32(1), sys1.PerPerson[1].QuestionsMeasured)

	// Per-question breakdown, first-seen order (q10 then q11).
	assert.Len(t, sys1.PerQuestion, 2)
	assert.Equal(t, int32(10), sys1.PerQuestion[0].QuestionID)
	assert.Equal(t, int32(2), sys1.PerQuestion[0].People, "A and B both touched q10")
	assert.Equal(t, float64(40), sys1.PerQuestion[0].AverageSecondsPerPerson, "(60+20)/2 people")
	assert.Equal(t, int32(11), sys1.PerQuestion[1].QuestionID)
	assert.Equal(t, int32(1), sys1.PerQuestion[1].People)
	assert.Equal(t, float64(30), sys1.PerQuestion[1].AverageSecondsPerPerson)

	sys2 := out[1]
	assert.Equal(t, int32(2), sys2.FismaSystemID)
	assert.Equal(t, float64(15), sys2.TotalSeconds)
	assert.Equal(t, float64(15), sys2.EditorSeconds)
	assert.Equal(t, float64(15), sys2.AverageSecondsPerQuestion)
	assert.Len(t, sys2.PerPerson, 1)
	assert.Len(t, sys2.PerQuestion, 1)
}

// TestRollupTimeSpentEmpty verifies no rows yields no systems (not a nil-deref).
func TestRollupTimeSpentEmpty(t *testing.T) {
	assert.Empty(t, rollupTimeSpent(nil))
}

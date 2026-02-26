package model

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildAggregateSubQuery mirrors the SQL building logic in FindScoresAggregate
// so we can test filter combinations without a database connection.
func buildAggregateSubQuery(input FindScoresInput) (string, []interface{}, error) {
	if input.FismaSystemID != nil && len(input.FismaSystemIDs) == 0 {
		input.FismaSystemIDs = []*int32{input.FismaSystemID}
	}

	subSqlb := squirrel.Select("datacallid, fismasystemid, AVG(score) OVER (PARTITION BY datacallid, fismasystemid) as systemscore").
		From("scores").
		InnerJoin("functionoptions on functionoptions.functionoptionid=scores.functionoptionid")

	if input.DataCallID != nil {
		subSqlb = subSqlb.Where("datacallid=?", input.DataCallID)
	}

	if len(input.FismaSystemIDs) > 0 {
		subSqlb = subSqlb.Where(squirrel.Eq{"fismasystemid": input.FismaSystemIDs})
	}

	if input.FismaSystemID != nil && len(input.FismaSystemIDs) >= 1 {
		subSqlb = subSqlb.Where("fismasystemid=?", *input.FismaSystemID)
	}

	return squirrel.Select("*").
		FromSelect(subSqlb, "avg_by_datacall_fismasystem").
		GroupBy("datacallid, fismasystemid, systemscore").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
}

func int32Ptr(i int32) *int32 { return &i }

// TestFindScoresAggregate_ISSOwithSpecificSystem is the regression test for the bug where
// an ISSO requesting a specific fismasystemid would get scores for ALL their assigned systems
// because the equality filter was skipped when FismaSystemIDs was pre-populated.
func TestFindScoresAggregate_ISSOwithSpecificSystem(t *testing.T) {
	sys1, sys2, sys3 := int32Ptr(1001), int32Ptr(1002), int32Ptr(1003)

	t.Run("ISSOwithMultipleSystemsRequestsSpecific", func(t *testing.T) {
		// Simulate controller setting FismaSystemIDs to the user's assigned systems,
		// then query param setting FismaSystemID to the specific requested system.
		input := FindScoresInput{
			FismaSystemIDs: []*int32{sys1, sys2, sys3},
			FismaSystemID:  sys1,
		}

		sql, args, err := buildAggregateSubQuery(input)
		require.NoError(t, err)

		// Must include the IN clause scoping to assigned systems
		assert.Contains(t, sql, "fismasystemid IN", "should scope to assigned systems")
		// Must also include an equality predicate (squirrel writes fismasystemid=$N, no spaces)
		assert.Contains(t, sql, "fismasystemid=$", "should have equality filter for specific system")
		// 3 args for IN list + 1 for equality = 4 total
		assert.Len(t, args, 4, "should have 4 args: 3 for IN list + 1 for equality")
	})

	t.Run("ISSOwithSingleSystemRequestsSpecific", func(t *testing.T) {
		// Edge case: ISSO with only one assigned system.
		input := FindScoresInput{
			FismaSystemIDs: []*int32{sys1},
			FismaSystemID:  sys1,
		}

		sql, args, err := buildAggregateSubQuery(input)
		require.NoError(t, err)

		assert.Contains(t, sql, "fismasystemid", "should filter on fismasystemid")
		assert.Contains(t, args, sys1)
	})

	t.Run("AdminRequestsSpecificSystem", func(t *testing.T) {
		// Admin path: FismaSystemIDs is empty, only FismaSystemID from query param.
		// The conversion block should promote FismaSystemID -> FismaSystemIDs.
		input := FindScoresInput{
			FismaSystemID: sys2,
		}

		sql, args, err := buildAggregateSubQuery(input)
		require.NoError(t, err)

		assert.Contains(t, sql, "fismasystemid", "should filter on fismasystemid")
		assert.Contains(t, args, sys2)
	})

	t.Run("ISSOwithNoSpecificSystem", func(t *testing.T) {
		// ISSO list view: no fismasystemid query param, just the assigned systems scope.
		// Should return all assigned systems (no equality filter added).
		input := FindScoresInput{
			FismaSystemIDs: []*int32{sys1, sys2},
		}

		sql, args, err := buildAggregateSubQuery(input)
		require.NoError(t, err)

		assert.Contains(t, sql, "fismasystemid IN", "should scope to assigned systems")
		// No equality filter when no specific system is requested
		assert.NotContains(t, sql, "fismasystemid=$", "should not have equality filter when no specific system requested")
		// 2 args for IN list only
		assert.Len(t, args, 2, "should have 2 args for IN list only")
	})
}

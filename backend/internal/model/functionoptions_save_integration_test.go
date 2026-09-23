package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctionOptionSaveIntegration covers the write path functionoptions gained
// in ztmf-misc#398. Before it, the 1-4 scoring scale every answer points at
// could only be changed by hand in SQL.
//
// Every option is created against a function this test makes itself, on the
// "AWS" environment, which no seeded system maps to - so a fixture is never
// visible to a questionnaire while the test runs, and the seeded functions keep
// exactly the four options the fixtures and screenshots expect.
// functionoptions.functionid is ON DELETE CASCADE, so deleting the fixture
// function is enough to clean up its options.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFunctionOptionSaveIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	questionID := int32(8004)

	newFixtureFunction := func(t *testing.T, name string) int32 {
		t.Helper()
		f := &Function{
			Function:              name,
			Description:           "functionoption save fixture",
			DataCenterEnvironment: "AWS",
			QuestionID:            &questionID,
		}
		saved, err := f.Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { hardDeleteFunctionByID(saved.FunctionID) })
		return saved.FunctionID
	}

	t.Run("CreatesAnOption", func(t *testing.T) {
		functionID := newFixtureFunction(t, "option create fixture")

		fo := &FunctionOption{
			FunctionID:  functionID,
			Score:       2,
			OptionName:  "Initial",
			Description: "some progress",
		}
		saved, err := fo.Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.NotZero(t, saved.FunctionOptionID, "the insert returns the generated id")
		assert.Equal(t, functionID, saved.FunctionID)
		assert.EqualValues(t, 2, saved.Score)

		got, err := FindFunctionOptionByID(ctx, saved.FunctionOptionID)
		require.NoError(t, err)
		assert.Equal(t, "Initial", got.OptionName)
		assert.Equal(t, "some progress", got.Description)
		assert.EqualValues(t, 2, got.Score)
	})

	t.Run("UpdatesAnOption", func(t *testing.T) {
		functionID := newFixtureFunction(t, "option update fixture")

		saved, err := (&FunctionOption{
			FunctionID: functionID, Score: 1, OptionName: "Traditional", Description: "d",
		}).Save(ctx)
		require.NoError(t, err)

		saved.Score = 4
		saved.OptionName = "Optimal"
		updated, err := saved.Save(ctx)
		require.NoError(t, err)
		assert.EqualValues(t, 4, updated.Score)

		got, err := FindFunctionOptionByID(ctx, saved.FunctionOptionID)
		require.NoError(t, err)
		assert.Equal(t, "Optimal", got.OptionName)
		assert.EqualValues(t, 4, got.Score, "and so does the persisted row")
	})

	t.Run("OffScaleScoreIsRejectedBeforeTheWrite", func(t *testing.T) {
		functionID := newFixtureFunction(t, "option offscale fixture")

		saved, err := (&FunctionOption{
			FunctionID: functionID, Score: 5, OptionName: "Beyond", Description: "d",
		}).Save(ctx)
		assert.Nil(t, saved)

		var invalid *InvalidInputError
		if assert.ErrorAs(t, err, &invalid) {
			assert.Contains(t, invalid.Data(), "score")
		}

		options, err := FindFunctionOptions(ctx, FindFunctionOptionsInput{FunctionID: &functionID})
		require.NoError(t, err)
		assert.Empty(t, options, "a rejected option must not reach the table")
	})

	t.Run("UnknownFunctionIsRejected", func(t *testing.T) {
		missing := int32(999999)
		saved, err := (&FunctionOption{
			FunctionID: missing, Score: 1, OptionName: "Traditional", Description: "d",
		}).Save(ctx)
		assert.Nil(t, saved)
		assert.ErrorIs(t, err, ErrNoReference, "the FK violation surfaces as a 400, not a 500")
	})

	t.Run("FindByIDRejectsANonPositiveID", func(t *testing.T) {
		got, err := FindFunctionOptionByID(ctx, 0)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, ErrNoData)
	})

	// The read path's ordering contract (ztmf-misc#279) has to survive an option
	// written out of score order, since nothing forces inserts to be sequential.
	t.Run("NewOptionsComeBackInScoreOrder", func(t *testing.T) {
		functionID := newFixtureFunction(t, "option ordering fixture")

		for _, score := range []int32{4, 1, 3, 2} {
			_, err := (&FunctionOption{
				FunctionID: functionID, Score: score, OptionName: "opt", Description: "d",
			}).Save(ctx)
			require.NoError(t, err)
		}

		options, err := FindFunctionOptions(ctx, FindFunctionOptionsInput{FunctionID: &functionID})
		require.NoError(t, err)
		require.Len(t, options, 4)
		for i, want := range []int32{1, 2, 3, 4} {
			assert.EqualValues(t, want, options[i].Score, "row %d", i)
		}
	})
}

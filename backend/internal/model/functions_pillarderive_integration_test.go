package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctionPillarDeriveIntegration proves a function's pillar is derived from
// its question on every write, so functions.pillarid can no longer disagree with
// questions.pillarid no matter what the caller sends.
//
// Fixtures come from the empire seed: question 8004 is in pillar 2 (Applications).
// The fixtures use the "AWS" environment, which no seeded system maps to, so a
// fixture is never visible to a questionnaire while the test runs. Requires DB_*
// env vars pointing at a seeded ZTMF database. Skipped under `go test -short`.
func TestFunctionPillarDeriveIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	questionID := int32(8004) // pillar 2

	t.Run("InsertIgnoresCallerPillar", func(t *testing.T) {
		f := &Function{
			Function:              "pillar derive insert fixture",
			Description:           "pillar must come from the question, not the caller",
			DataCenterEnvironment: "AWS",
			Ordr:                  99,
			QuestionID:            &questionID,
			PillarID:              1, // deliberately wrong
		}
		saved, err := f.Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, saved)
		t.Cleanup(func() { hardDeleteFunctionByID(saved.FunctionID) })

		assert.Equal(t, int32(2), saved.PillarID, "returned struct carries the question's pillar")

		got, err := FindFunctionByID(ctx, saved.FunctionID)
		require.NoError(t, err)
		assert.Equal(t, int32(2), got.PillarID, "and so does the persisted row")
	})

	t.Run("UpdateIgnoresCallerPillar", func(t *testing.T) {
		f := &Function{
			Function:              "pillar derive update fixture",
			Description:           "an admin edit must not be able to introduce drift",
			DataCenterEnvironment: "AWS",
			Ordr:                  99,
			QuestionID:            &questionID,
			PillarID:              2,
		}
		saved, err := f.Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { hardDeleteFunctionByID(saved.FunctionID) })

		saved.PillarID = 6 // deliberately wrong
		updated, err := saved.Save(ctx)
		require.NoError(t, err)
		assert.Equal(t, int32(2), updated.PillarID)

		got, err := FindFunctionByID(ctx, saved.FunctionID)
		require.NoError(t, err)
		assert.Equal(t, int32(2), got.PillarID)
	})

	t.Run("MissingQuestionIsRejected", func(t *testing.T) {
		f := &Function{
			Function:              "pillar derive questionless fixture",
			Description:           "must not be written",
			DataCenterEnvironment: "AWS",
			Ordr:                  99,
			PillarID:              3,
		}
		saved, err := f.Save(ctx)
		assert.Nil(t, saved)

		var invalid *InvalidInputError
		if assert.ErrorAs(t, err, &invalid) {
			assert.Contains(t, invalid.Data(), "questionid")
		}
	})

	t.Run("UnknownQuestionIsRejected", func(t *testing.T) {
		missing := int32(999999)
		f := &Function{
			Function:              "pillar derive bad question fixture",
			Description:           "must not be written",
			DataCenterEnvironment: "AWS",
			Ordr:                  99,
			QuestionID:            &missing,
			PillarID:              1,
		}
		saved, err := f.Save(ctx)
		assert.ErrorIs(t, err, ErrNoReference, "same 400 the FK violation produced, caught before the insert")
		assert.Nil(t, saved)
	})

	t.Run("SeededDataAgrees", func(t *testing.T) {
		c, err := db.Conn(ctx)
		require.NoError(t, err)
		defer c.Release()

		var drifted int
		err = c.QueryRow(ctx, `
			SELECT count(*)
			  FROM functions f
			  JOIN questions q ON q.questionid = f.questionid
			 WHERE f.pillarid <> q.pillarid`).Scan(&drifted)
		require.NoError(t, err)
		assert.Zero(t, drifted, "no function may disagree with its question's pillar")
	})
}

func hardDeleteFunctionByID(id int32) {
	c, err := db.Conn(context.Background())
	if err != nil {
		return
	}
	defer c.Release()
	_, _ = c.Exec(context.Background(), `DELETE FROM functions WHERE functionid = $1`, id)
}

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Question.validate had been a commented-out stub since the endpoint shipped,
// and Save called no validation at all, so an empty POST inserted a row with
// empty text and pillarid 0 - which only failed later, on the FK, as a
// reference error naming no field (ztmf-misc#398).
func TestQuestionValidate(t *testing.T) {
	valid := func() *Question {
		return &Question{
			Question:    "How do you manage user identities?",
			NotesPrompt: "Describe your approach",
			PillarID:    1,
		}
	}

	t.Run("accepts a complete question", func(t *testing.T) {
		assert.NoError(t, valid().validate())
	})

	t.Run("accepts a complete question with no order", func(t *testing.T) {
		// ordr is a *int so a PUT omitting "order" preserves the stored rank;
		// requiring it here would defeat that.
		q := valid()
		q.Ordr = nil
		assert.NoError(t, q.validate())
	})

	t.Run("accepts an explicit order", func(t *testing.T) {
		ordr := 101
		q := valid()
		q.Ordr = &ordr
		assert.NoError(t, q.validate())
	})

	fields := []struct {
		name   string
		mutate func(*Question)
		key    string
	}{
		{"empty question text", func(q *Question) { q.Question = "" }, "question"},
		{"empty notesprompt", func(q *Question) { q.NotesPrompt = "" }, "notesprompt"},
		{"whitespace-only question text", func(q *Question) { q.Question = "   " }, "question"},
		{"whitespace-only notesprompt", func(q *Question) { q.NotesPrompt = "\t\n " }, "notesprompt"},
		{"zero pillarid", func(q *Question) { q.PillarID = 0 }, "pillarid"},
		{"negative pillarid", func(q *Question) { q.PillarID = -1 }, "pillarid"},
	}

	for _, f := range fields {
		t.Run("rejects "+f.name, func(t *testing.T) {
			q := valid()
			f.mutate(q)

			err := q.validate()

			var invalid *InvalidInputError
			require.ErrorAs(t, err, &invalid)
			assert.Contains(t, invalid.Data(), f.key)
			assert.Len(t, invalid.Data(), 1, "only the offending field is reported")
		})
	}

	t.Run("reports every offending field at once", func(t *testing.T) {
		err := (&Question{}).validate()

		var invalid *InvalidInputError
		require.ErrorAs(t, err, &invalid)
		assert.Contains(t, invalid.Data(), "question")
		assert.Contains(t, invalid.Data(), "notesprompt")
		assert.Contains(t, invalid.Data(), "pillarid")
	})

	// A plain int falls through isValidIntID's int32/*int32 type switch and
	// reports false for every value, so using it here would have rejected every
	// question ever submitted. Pins the explicit comparison instead.
	t.Run("a valid pillarid is not rejected by the int/int32 trap", func(t *testing.T) {
		for _, id := range []int{1, 2, 6, 99} {
			q := valid()
			q.PillarID = id
			assert.NoErrorf(t, q.validate(), "pillarid %d must be accepted", id)
		}
	})
}

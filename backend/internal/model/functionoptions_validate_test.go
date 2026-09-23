package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FunctionOption gained its first write path in ztmf-misc#398; before that the
// 1-4 scoring scale every answer points at could only be changed by hand in SQL.
func TestFunctionOptionValidate(t *testing.T) {
	valid := func() *FunctionOption {
		return &FunctionOption{
			FunctionID:  7001,
			Score:       3,
			OptionName:  "Advanced",
			Description: "Continuous verification",
		}
	}

	t.Run("accepts a complete option", func(t *testing.T) {
		assert.NoError(t, valid().validate())
	})

	t.Run("accepts an empty description", func(t *testing.T) {
		// functionoptions.description is nullable, unlike functions.description.
		fo := valid()
		fo.Description = ""
		assert.NoError(t, fo.validate())
	})

	t.Run("accepts every score on the maturity scale", func(t *testing.T) {
		for score := int32(1); score <= 4; score++ {
			fo := valid()
			fo.Score = score
			assert.NoErrorf(t, fo.validate(), "score %d is on the scale", score)
		}
	})

	t.Run("accepts an optionname exactly at the column limit", func(t *testing.T) {
		fo := valid()
		fo.OptionName = strings.Repeat("a", maxOptionNameLen)
		assert.NoError(t, fo.validate())
	})

	cases := []struct {
		name   string
		mutate func(*FunctionOption)
		key    string
	}{
		{"score below the scale", func(fo *FunctionOption) { fo.Score = 0 }, "score"},
		{"score above the scale", func(fo *FunctionOption) { fo.Score = 5 }, "score"},
		{"negative score", func(fo *FunctionOption) { fo.Score = -1 }, "score"},
		{"empty optionname", func(fo *FunctionOption) { fo.OptionName = "" }, "optionname"},
		{"optionname past the column limit", func(fo *FunctionOption) {
			fo.OptionName = strings.Repeat("a", maxOptionNameLen+1)
		}, "optionname"},
		{"missing functionid", func(fo *FunctionOption) { fo.FunctionID = 0 }, "functionid"},
	}

	for _, c := range cases {
		t.Run("rejects "+c.name, func(t *testing.T) {
			fo := valid()
			c.mutate(fo)

			err := fo.validate()

			var invalid *InvalidInputError
			require.ErrorAs(t, err, &invalid)
			assert.Contains(t, invalid.Data(), c.key)
			assert.Len(t, invalid.Data(), 1, "only the offending field is reported")
		})
	}

	// Four-options-per-function with no duplicate score is a publish-time rule
	// (ztmf-misc#396's gate), deliberately not enforced on write: a set cannot
	// be edited if an intermediate state is rejected, and #398 ships no DELETE.
	t.Run("does not enforce the per-function set invariant", func(t *testing.T) {
		fo := valid()
		fo.Score = 3
		assert.NoError(t, fo.validate(), "a duplicate score is a publish concern, not a write concern")
	})
}

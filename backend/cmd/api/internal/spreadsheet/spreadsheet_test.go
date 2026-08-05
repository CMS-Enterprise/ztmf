package spreadsheet

import (
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }

// TestExcelRendersNotStartedBlankVsRealZero pins the #526 distinction in the
// output cells: a function the system never answered (nil option/score/notes)
// renders as blank, while a function genuinely answered at score 0 renders "0".
// The two must read differently, which is the whole point of carrying the answer
// fields as nullable rather than coalescing an absent answer to zero.
func TestExcelRendersNotStartedBlankVsRealZero(t *testing.T) {
	answers := []*model.Answer{
		{ // row 2: applicable function, never answered
			FismaAcronym:          "SYS-NOTSTARTED",
			DataCenterEnvironment: "Env",
			Pillar:                "Identity",
			Function:              "Fn",
			Description:           "Desc",
			Question:              "Q",
			OptionDescription:     nil,
			OptionName:            nil,
			Score:                 nil,
			Notes:                 nil,
		},
		{ // row 3: answered at a genuine zero
			FismaAcronym:          "SYS-ZERO",
			DataCenterEnvironment: "Env",
			Pillar:                "Identity",
			Function:              "Fn",
			Description:           "Desc",
			Question:              "Q",
			OptionDescription:     strptr("Lowest tier"),
			OptionName:            strptr("Traditional"),
			Score:                 intptr(0),
			Notes:                 strptr("some note"),
		},
	}

	f, err := Excel(answers)
	require.NoError(t, err)

	get := func(cell string) string {
		v, err := f.GetCellValue("Sheet1", cell)
		require.NoError(t, err)
		return v
	}

	// Not-started row: answer, tier, score, and notes cells are all blank.
	assert.Equal(t, "", get("G2"), "not-started answer cell should be blank")
	assert.Equal(t, "", get("H2"), "not-started tier cell should be blank")
	assert.Equal(t, "", get("I2"), "not-started score cell should be blank, not 0")
	assert.Equal(t, "", get("J2"), "not-started notes cell should be blank")

	// Real-zero row: the score renders 0, distinguishable from the blank above.
	assert.Equal(t, "Lowest tier", get("G3"))
	assert.Equal(t, "Traditional", get("H3"))
	assert.Equal(t, "0", get("I3"), "a genuine zero score must render 0, not blank")
	assert.Equal(t, "some note", get("J3"))
}

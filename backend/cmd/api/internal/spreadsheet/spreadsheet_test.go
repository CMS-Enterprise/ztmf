package spreadsheet

import (
	"fmt"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }

// TestExcelRendersNotStartedBlankVsAnswered pins the #526 distinction in the
// output cells: a function the system never answered (nil option/score/notes)
// renders as blank, while a function answered at the lowest option (score 1,
// answers are 1-4) renders "1". The two must read differently, which is the
// whole point of carrying the answer fields as nullable rather than coalescing
// an absent answer to a number.
func TestExcelRendersNotStartedBlankVsAnswered(t *testing.T) {
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
		{ // row 3: answered at the lowest option
			FismaAcronym:          "SYS-LOWEST",
			DataCenterEnvironment: "Env",
			Pillar:                "Identity",
			Function:              "Fn",
			Description:           "Desc",
			Question:              "Q",
			OptionDescription:     strptr("Lowest tier"),
			OptionName:            strptr("Traditional"),
			Score:                 intptr(1),
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

	// Answered row: the score renders 1, distinguishable from the blank above.
	assert.Equal(t, "Lowest tier", get("G3"))
	assert.Equal(t, "Traditional", get("H3"))
	assert.Equal(t, "1", get("I3"), "an answered score must render its value, not blank")
	assert.Equal(t, "some note", get("J3"))
}

// TestExcelLabelsScoreAndExplainsFormula pins the ztmf-misc#383 ask: the raw
// answer column says it is on the 1-4 scale, and a second sheet states how the
// dashboard builds its scores from it.
func TestExcelLabelsScoreAndExplainsFormula(t *testing.T) {
	f, err := Excel(nil)
	require.NoError(t, err)

	header, err := f.GetCellValue("Sheet1", "I1")
	require.NoError(t, err)
	assert.Equal(t, "Answer Score (1-4)", header)

	assert.Contains(t, f.GetSheetList(), scoringNotesSheet)
	for i, note := range scoringNotes {
		got, err := f.GetCellValue(scoringNotesSheet, fmt.Sprintf("A%d", i+1))
		require.NoError(t, err)
		assert.Equal(t, note, got)
	}
}

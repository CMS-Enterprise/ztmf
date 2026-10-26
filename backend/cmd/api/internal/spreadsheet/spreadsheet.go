package spreadsheet

import (
	"fmt"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/xuri/excelize/v2"
)

func Excel(answers []*model.Answer) (*excelize.File, error) {

	sheet := "Sheet1"

	f := excelize.NewFile()

	f.SetCellValue(sheet, "A1", "Fisma Acronym")
	f.SetCellValue(sheet, "B1", "Data Center Environment")
	f.SetCellValue(sheet, "C1", "Pillar")
	f.SetCellValue(sheet, "D1", "Function")
	f.SetCellValue(sheet, "E1", "Function Description")
	f.SetCellValue(sheet, "F1", "Question")
	f.SetCellValue(sheet, "G1", "Answer")
	f.SetCellValue(sheet, "H1", "Maturity Tier")
	f.SetCellValue(sheet, "I1", "Answer Score (1-4)")
	f.SetCellValue(sheet, "J1", "ADO Answer Details")
	f.SetCellValue(sheet, "K1", "Target Maturity Level")
	f.SetCellValue(sheet, "L1", "Target Justification")

	for i, a := range answers {
		row := i + 2 // i starts at 0 and headers are in row 1
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), a.FismaAcronym)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), a.DataCenterEnvironment)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), a.Pillar)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), a.Function)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), a.Description)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), a.Question)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), derefString(a.OptionDescription))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), derefString(a.OptionName))
		if a.Score != nil {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), *a.Score)
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), derefString(a.Notes))
		// NULL target = no explicit assertion yet; the app-wide default is
		// Advanced, and the export says so rather than leaving readers to guess.
		targetTier := "Advanced (default)"
		targetJustification := ""
		if a.TargetMaturityTier != nil {
			targetTier = *a.TargetMaturityTier
		}
		if a.TargetMaturityJustification != nil {
			targetJustification = *a.TargetMaturityJustification
		}
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), targetTier)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), targetJustification)
	}

	if err := addScoringNotes(f); err != nil {
		return nil, err
	}

	return f, nil
}

// scoringNotesSheet explains how the dashboard turns the raw 1-4 answers in
// column I into scores, so an analyst can reproduce it (ztmf-misc#383, #433).
const scoringNotesSheet = "How scores work"

var scoringNotes = []string{
	"Column I is the raw answer score: 1 Traditional, 2 Initial, 3 Advanced, 4 Optimal. A blank row is a question the system has not answered.",
	"Dashboard pillar score = average of the pillar's answer scores x 1.25 (the HHS scorecard formula, sum x 5 / maximum). An unanswered question counts as 1.",
	"A pillar or system with nothing answered is labeled Not Assessed, but still scores 1.25 and counts in averages.",
	"Dashboard system score = average of the pillar scores, each pillar weighted equally.",
	"Maturity tiers: Optimal 4.10 and up, Advanced 3.10, Initial 2.10, Traditional 1.01.",
	"Averaging column I directly will not match the dashboard: it skips blanks, does not apply x 1.25, and weights pillars by their question count.",
}

func addScoringNotes(f *excelize.File) error {
	if _, err := f.NewSheet(scoringNotesSheet); err != nil {
		return err
	}
	for i, note := range scoringNotes {
		if err := f.SetCellValue(scoringNotesSheet, fmt.Sprintf("A%d", i+1), note); err != nil {
			return err
		}
	}
	return nil
}

// derefString returns the pointed-to string, or "" when the pointer is nil, so a
// not-started answer renders as a blank cell rather than a zero value.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

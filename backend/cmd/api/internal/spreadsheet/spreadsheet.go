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
	f.SetCellValue(sheet, "I1", "Score")
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
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), a.OptionDescription)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), a.OptionName)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), a.Score)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), a.Notes)
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

	return f, nil
}

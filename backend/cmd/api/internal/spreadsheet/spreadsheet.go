package spreadsheet

import (
	"fmt"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/xuri/excelize/v2"
)

func Excel(answers []*model.Answer) (*excelize.File, error) {

	sheet := "Sheet1"

	f := excelize.NewFile()

	f.SetCellValue(sheet, "A1", "Fisma Acronym")
	f.SetCellValue(sheet, "B1", "Data Center Environment")
	f.SetCellValue(sheet, "C1", "Pillar")
	f.SetCellValue(sheet, "D1", "Function")
	f.SetCellValue(sheet, "E1", "Question")
	f.SetCellValue(sheet, "F1", "Description")
	f.SetCellValue(sheet, "G1", "Answer")
	f.SetCellValue(sheet, "H1", "Score")
	f.SetCellValue(sheet, "I1", "Notes")

	for i, a := range answers {
		row := i + 2 // i starts at 0 and headers are in row 1
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), a.FismaAcronym)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), a.DataCenterEnvironment)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), a.Pillar)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), a.Function)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), a.Question)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), a.Description)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), a.OptionName)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), a.Score)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), a.Notes)
	}

	return f, nil
}

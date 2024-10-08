package spreadsheet

import (
	"fmt"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/xuri/excelize/v2"
)

func Excel(answers []*model.Answer) (*excelize.File, error) {

	var (
		sheet   = "Scores"
		headers = []string{
			"Data Call",
			"Fisma Acronym",
			"Data Center Environment",
			"Pillar",
			"Question",
			"Function",
			"Description",
			"Answer",
			"Score",
			"Notes",
		}
	)

	f := excelize.NewFile()
	i, err := f.NewSheet(sheet)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	f.SetActiveSheet(i)
	for i, v := range headers {
		col := rune(int32('A') + int32(i))
		f.SetCellValue(sheet, fmt.Sprintf("%s1", string(col)), v)
	}

	for i, a := range answers {
		row := i + 2 // headers are in row 1
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), a.DataCall)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), a.FismaAcronym)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), a.DataCenterEnvironment)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), a.Pillar)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), a.Question)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), a.Function)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), a.Description)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), a.OptionName)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), a.Score)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), a.Notes)
	}

	return f, nil
}

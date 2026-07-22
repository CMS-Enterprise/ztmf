package spreadsheet

import (
	"fmt"
	"math"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/xuri/excelize/v2"
)

// SystemInfo carries the display fields for a FISMA system that the time-spent
// export needs but the TimeSpent model (keyed only by id) does not. The
// controller resolves these once and passes them in so this package stays
// database-free, mirroring how Excel() takes pre-fetched answers.
type SystemInfo struct {
	Acronym string
	Name    string
}

// timeSpentNote is the header banner on every sheet: these figures reflect only
// activity recorded since time tracking was enabled, and are a lower bound (each
// interval is capped at the idle limit and a person's final action contributes
// no time).
const timeSpentNote = "Figures reflect questionnaire activity recorded since time tracking was enabled; data calls with no recorded activity are blank. They are a lower bound: each interval is capped at the idle limit and a person's final action contributes no time."

// noActivityStatus is the Status-column value for a system with nothing recorded
// this data call, shown on all three sheets so they list the same systems.
const noActivityStatus = "No activity recorded"

// TimeSpentExcel builds the 3-sheet time-spent workbook for a data call:
// System Totals, Per Person, and Per Question. dataCall is the human-readable
// data call label shown on each sheet so the export is self-identifying. systems
// maps a FISMA system id to its display acronym/name and questions maps a
// question id to its text; a missing entry falls back to the numeric id so the
// export never blanks a row.
func TimeSpentExcel(timeSpent []*model.TimeSpent, systems map[int32]SystemInfo, questions map[int32]string, dataCall string) (*excelize.File, error) {
	f := excelize.NewFile()

	acronym := func(id int32) string {
		if si, ok := systems[id]; ok && si.Acronym != "" {
			return si.Acronym
		}
		return fmt.Sprintf("%d", id)
	}
	name := func(id int32) string {
		if si, ok := systems[id]; ok {
			return si.Name
		}
		return ""
	}
	questionText := func(id int32) string {
		if q, ok := questions[id]; ok {
			return q
		}
		return fmt.Sprintf("%d", id)
	}

	// A shared wrap style so long text (the note banner, question text) shows in
	// full inside its cell rather than being clipped. NewStyle only errors on a
	// malformed style, so a zero id (no wrap) is an acceptable fallback.
	wrapStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
	})

	writeSystemTotals(f, timeSpent, dataCall, wrapStyle, acronym, name)
	writePerPerson(f, timeSpent, dataCall, wrapStyle, acronym)
	writePerQuestion(f, timeSpent, dataCall, wrapStyle, acronym, questionText)

	// Size every column to its widest header/data cell so content is not cut
	// off (excelize has no true auto-fit).
	for _, sheet := range []string{"System Totals", "Per Person", "Per Question"} {
		autoWidth(f, sheet)
	}

	// excelize seeds a default "Sheet1"; the first builder renames it, so no
	// stray empty sheet is left behind.
	return f, nil
}

// autoWidth sizes each column to its widest cell from the header row down,
// clamped to a sane range. Rows 1-2 (the data call title and the explanatory
// note) are skipped on purpose: they are long banners meant to overflow across
// the empty cells beside them, and sizing column A to the note would make it
// absurdly wide.
func autoWidth(f *excelize.File, sheet string) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return
	}
	const headerRow = 3 // 1-based; rows[headerRow-1] is the header
	widths := map[int]int{}
	for r := headerRow - 1; r < len(rows); r++ {
		for c, val := range rows[r] {
			if l := len([]rune(val)); l > widths[c] {
				widths[c] = l
			}
		}
	}
	for c, w := range widths {
		col, err := excelize.ColumnNumberToName(c + 1)
		if err != nil {
			continue
		}
		width := float64(w) + 2 // a little padding beyond the content
		if width < 12 {
			width = 12
		}
		if width > 80 { // cap so a long question/name does not run off-screen
			width = 80
		}
		f.SetColWidth(sheet, col, col, width)
	}
}

// minutes converts seconds to whole minutes, rounded, for a readable figure.
func minutes(seconds float64) int {
	return int(math.Round(seconds / 60.0))
}

// firstDataRow is the row where a sheet's data begins: row 1 is the data call
// title, row 2 the explanatory note, row 3 the column headers.
const firstDataRow = 4

// writeSheetHeader writes the shared top-of-sheet block (data call title, note,
// column headers) so every sheet is self-identifying and laid out identically.
// The note is merged across the sheet's columns and wrapped so it reads as a
// full paragraph; merged cells do not auto-height in Excel, so the row height is
// set explicitly to fit the wrapped text.
func writeSheetHeader(f *excelize.File, sheet, dataCall string, wrapStyle int, headers []string) {
	f.SetCellValue(sheet, "A1", fmt.Sprintf("Data Call: %s", dataCall))
	f.SetCellValue(sheet, "A2", timeSpentNote)

	lastCol, err := excelize.ColumnNumberToName(len(headers))
	if err == nil {
		f.MergeCell(sheet, "A2", fmt.Sprintf("%s2", lastCol))
		f.SetCellStyle(sheet, "A2", fmt.Sprintf("%s2", lastCol), wrapStyle)
		f.SetRowHeight(sheet, 2, 30)
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheet, cell, h)
	}
}

func writeSystemTotals(f *excelize.File, timeSpent []*model.TimeSpent, dataCall string, wrapStyle int, acronym, name func(int32) string) {
	sheet := "System Totals"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"System", "System Name", "Editor Minutes", "Viewer Minutes", "Total Minutes", "Questions", "Avg Seconds / Question", "Status"}
	writeSheetHeader(f, sheet, dataCall, wrapStyle, headers)

	for i, ts := range timeSpent {
		row := i + firstDataRow
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acronym(ts.FismaSystemID))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), name(ts.FismaSystemID))
		// A system nobody has opened this data call yet: name it and mark the
		// status, but leave the metric cells blank rather than a wall of 0s -
		// consistent with the Per Person / Per Question sheets, where a
		// no-activity system is a name + status with no numbers.
		if ts.QuestionsMeasured == 0 {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), noActivityStatus)
			continue
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), minutes(ts.EditorSeconds))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), minutes(ts.ViewerSeconds))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), minutes(ts.TotalSeconds))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), ts.QuestionsMeasured)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), math.Round(ts.AverageSecondsPerQuestion))
	}
}

func writePerPerson(f *excelize.File, timeSpent []*model.TimeSpent, dataCall string, wrapStyle int, acronym func(int32) string) {
	sheet := "Per Person"
	f.NewSheet(sheet)

	headers := []string{"System", "Name", "Email", "Role", "Editor Minutes", "Viewer Minutes", "Total Minutes", "Questions", "Status"}
	writeSheetHeader(f, sheet, dataCall, wrapStyle, headers)

	row := firstDataRow
	for _, ts := range timeSpent {
		// Mirror the System Totals sheet: a system with no recorded activity
		// still gets a named row here (with an explicit status) rather than
		// vanishing, so all three sheets list the same systems.
		if len(ts.PerPerson) == 0 {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acronym(ts.FismaSystemID))
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), noActivityStatus)
			row++
			continue
		}
		for _, p := range ts.PerPerson {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acronym(ts.FismaSystemID))
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.Name)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.Email)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.Role)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), minutes(p.EditorSeconds))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), minutes(p.ViewerSeconds))
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), minutes(p.TotalSeconds))
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.QuestionsMeasured)
			row++
		}
	}
}

func writePerQuestion(f *excelize.File, timeSpent []*model.TimeSpent, dataCall string, wrapStyle int, acronym func(int32) string, questionText func(int32) string) {
	sheet := "Per Question"
	f.NewSheet(sheet)

	headers := []string{"System", "Question", "Avg Seconds / Person", "People", "Status"}
	writeSheetHeader(f, sheet, dataCall, wrapStyle, headers)

	row := firstDataRow
	for _, ts := range timeSpent {
		// As above: keep a named row for a system with no recorded activity so
		// this sheet lists the same systems as System Totals.
		if len(ts.PerQuestion) == 0 {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acronym(ts.FismaSystemID))
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), noActivityStatus)
			row++
			continue
		}
		for _, q := range ts.PerQuestion {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acronym(ts.FismaSystemID))
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), questionText(q.QuestionID))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), math.Round(q.AverageSecondsPerPerson))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), q.People)
			// Question text is the one free-text column that can run long; wrap it
			// so it stays fully readable within the 80-char-capped column.
			f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), wrapStyle)
			row++
		}
	}
}

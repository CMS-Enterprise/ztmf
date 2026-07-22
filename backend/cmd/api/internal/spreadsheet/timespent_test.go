package spreadsheet

import (
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimeSpentExcel_ZeroActivitySystemStillListed pins the fix for a system
// nobody has opened yet (issue #368 follow-up): the export must still show the
// system's identity and an explicit "No activity recorded" status rather than a
// header-only sheet, so the user knows the export succeeded and simply has no
// data. The per-person and per-question sheets, having no rows, carry a
// placeholder line to the same end.
func TestTimeSpentExcel_ZeroActivitySystemStillListed(t *testing.T) {
	timeSpent := []*model.TimeSpent{
		// A requested system with no recorded activity: zeroed metrics, empty
		// breakdowns - exactly what the controller synthesizes for a system
		// missing from the query result.
		{FismaSystemID: 42},
	}
	systems := map[int32]SystemInfo{
		42: {Acronym: "SSD-EX", Name: "Super Star Destroyer Executor"},
	}

	f, err := TimeSpentExcel(timeSpent, systems, map[int32]string{}, "FY2026_Q1")
	require.NoError(t, err)

	// Every sheet is self-identifying with the data call in the title row.
	title, err := f.GetCellValue("System Totals", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Data Call: FY2026_Q1", title, "the data call must be named on the sheet")

	// System Totals: the system's identity is present and the status is explicit
	// (data begins on row 4: title, note, headers occupy rows 1-3).
	acronym, err := f.GetCellValue("System Totals", "A4")
	require.NoError(t, err)
	assert.Equal(t, "SSD-EX", acronym, "the system acronym must appear even with no activity")

	name, err := f.GetCellValue("System Totals", "B4")
	require.NoError(t, err)
	assert.Equal(t, "Super Star Destroyer Executor", name)

	status, err := f.GetCellValue("System Totals", "H4")
	require.NoError(t, err)
	assert.Equal(t, "No activity recorded", status)

	// The Per Person and Per Question sheets list the same system by name with an
	// explicit status, matching System Totals, rather than dropping it.
	perPersonSys, err := f.GetCellValue("Per Person", "A4")
	require.NoError(t, err)
	assert.Equal(t, "SSD-EX", perPersonSys, "no-activity system is still named on Per Person")
	perPersonStatus, err := f.GetCellValue("Per Person", "I4") // Status is the 9th column
	require.NoError(t, err)
	assert.Equal(t, "No activity recorded", perPersonStatus)

	perQuestionSys, err := f.GetCellValue("Per Question", "A4")
	require.NoError(t, err)
	assert.Equal(t, "SSD-EX", perQuestionSys, "no-activity system is still named on Per Question")
	perQuestionStatus, err := f.GetCellValue("Per Question", "E4") // Status is the 5th column
	require.NoError(t, err)
	assert.Equal(t, "No activity recorded", perQuestionStatus)

	// The System Name column is auto-sized to fit its content ("Super Star
	// Destroyer Executor", 29 chars) rather than left at the ~9-wide default.
	nameWidth, err := f.GetColWidth("System Totals", "B")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nameWidth, float64(29), "columns should be widened to fit their content")
}

// TestTimeSpentExcel_PopulatedSystemRendersMetrics verifies a system WITH
// recorded activity lands its numbers in the right cells on every sheet: minutes
// on System Totals / Per Person, the average and question text on Per Question,
// and a blank Status (activity present).
func TestTimeSpentExcel_PopulatedSystemRendersMetrics(t *testing.T) {
	timeSpent := []*model.TimeSpent{
		{
			FismaSystemID:             5,
			EditorSeconds:             120, // 2 min
			ViewerSeconds:             60,  // 1 min
			TotalSeconds:              180, // 3 min
			QuestionsMeasured:         2,
			AverageSecondsPerQuestion: 90,
			PerPerson: []*model.TimeSpentPerson{
				{
					AuditRef:          model.AuditRef{UserID: "u-1", Name: "Alice", Email: "a@x", Role: "ISSO"},
					EditorSeconds:     120,
					ViewerSeconds:     60,
					TotalSeconds:      180,
					QuestionsMeasured: 2,
				},
			},
			PerQuestion: []*model.TimeSpentQuestion{
				{QuestionID: 900, AverageSecondsPerPerson: 90, People: 1},
			},
		},
	}
	systems := map[int32]SystemInfo{5: {Acronym: "SYS5", Name: "System Five"}}
	questions := map[int32]string{900: "How mature is access control?"}

	f, err := TimeSpentExcel(timeSpent, systems, questions, "FY2026_Q1")
	require.NoError(t, err)

	get := func(sheet, cell string) string {
		v, e := f.GetCellValue(sheet, cell)
		require.NoError(t, e)
		return v
	}

	// System Totals: minutes are rounded from seconds; status blank when active.
	assert.Equal(t, "SYS5", get("System Totals", "A4"))
	assert.Equal(t, "2", get("System Totals", "C4"), "editor 120s -> 2 min")
	assert.Equal(t, "1", get("System Totals", "D4"), "viewer 60s -> 1 min")
	assert.Equal(t, "3", get("System Totals", "E4"), "total 180s -> 3 min")
	assert.Equal(t, "2", get("System Totals", "F4"), "questions measured")
	assert.Equal(t, "90", get("System Totals", "G4"), "avg seconds per question")
	assert.Equal(t, "", get("System Totals", "H4"), "status is blank for an active system")

	// Per Person: the editor's row with minutes and a blank status.
	assert.Equal(t, "SYS5", get("Per Person", "A4"))
	assert.Equal(t, "Alice", get("Per Person", "B4"))
	assert.Equal(t, "3", get("Per Person", "G4"), "person total 180s -> 3 min")
	assert.Equal(t, "", get("Per Person", "I4"), "status blank for an active system")

	// Per Question: the question text, per-person average, and people count.
	assert.Equal(t, "SYS5", get("Per Question", "A4"))
	assert.Equal(t, "How mature is access control?", get("Per Question", "B4"))
	assert.Equal(t, "90", get("Per Question", "C4"))
	assert.Equal(t, "1", get("Per Question", "D4"))
	assert.Equal(t, "", get("Per Question", "E4"), "status blank for an active system")
}

// TestTimeSpentExcel_UnknownSystemFallsBackToID verifies a system missing from
// the info lookup renders its numeric id rather than blanking the row.
func TestTimeSpentExcel_UnknownSystemFallsBackToID(t *testing.T) {
	f, err := TimeSpentExcel([]*model.TimeSpent{{FismaSystemID: 7}}, map[int32]SystemInfo{}, map[int32]string{}, "FY2026_Q1")
	require.NoError(t, err)

	acronym, err := f.GetCellValue("System Totals", "A4")
	require.NoError(t, err)
	assert.Equal(t, "7", acronym)
}

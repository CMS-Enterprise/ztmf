package controller

import (
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/spreadsheet"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectExportSystems pins the time-spent export's "list every requested,
// visible system" assembly: results are reused where present, a zeroed row is
// synthesized where a system had no activity, out-of-scope ids are dropped, and
// ordering follows the request (or ascending id when unfiltered).
func TestSelectExportSystems(t *testing.T) {
	visible := map[int32]spreadsheet.SystemInfo{
		1: {Acronym: "SYS1"},
		2: {Acronym: "SYS2"},
	}
	withData := &model.TimeSpent{FismaSystemID: 1, QuestionsMeasured: 3, TotalSeconds: 120}

	t.Run("FsidsOrderReusesDataAndZeroFills", func(t *testing.T) {
		// Request 2 (no data), 1 (has data), 3 (out of scope). Order preserved,
		// 3 dropped, 2 synthesized as a zero row, 1 reused as-is.
		out := selectExportSystems([]*model.TimeSpent{withData}, visible, []int32{2, 1, 3})

		require.Len(t, out, 2, "out-of-scope id 3 is dropped")
		assert.Equal(t, int32(2), out[0].FismaSystemID)
		assert.Equal(t, int32(0), out[0].QuestionsMeasured, "system 2 is a synthesized zero row")
		assert.Equal(t, int32(1), out[1].FismaSystemID)
		assert.Same(t, withData, out[1], "system 1 reuses the analytics result, not a copy")
	})

	t.Run("NoFsidsListsAllVisibleSortedByID", func(t *testing.T) {
		out := selectExportSystems([]*model.TimeSpent{withData}, visible, nil)

		require.Len(t, out, 2)
		assert.Equal(t, int32(1), out[0].FismaSystemID, "ascending id order when unfiltered")
		assert.Equal(t, int32(2), out[1].FismaSystemID)
		assert.Equal(t, int32(0), out[1].QuestionsMeasured, "system 2 has no data -> zero row")
	})

	t.Run("DataForOutOfScopeSystemIsDropped", func(t *testing.T) {
		// Belt-and-suspenders: even if a result carried a system the caller
		// cannot see, requesting it must not surface it (it is not in visible).
		hidden := &model.TimeSpent{FismaSystemID: 99, QuestionsMeasured: 5}
		out := selectExportSystems([]*model.TimeSpent{hidden}, visible, []int32{99})

		assert.Empty(t, out, "a system absent from the visible set is never exported")
	})
}

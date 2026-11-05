package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The rollover between calls on one version is the live path every annual
// cycle takes; ztmf-misc#397 changes it only by the trailing ON CONFLICT.
func TestRolloverSQLSameVersionIsUnchanged(t *testing.T) {
	sql, args := rolloverSQL(41, 42, 1, false)
	assert.Equal(t, "INSERT INTO scores (fismasystemid,datecalculated,notes,notes_is_ai_summary,functionoptionid,datacallid,status) SELECT DISTINCT ON (s.fismasystemid, fo.functionid) s.fismasystemid, s.datecalculated, s.notes, s.notes_is_ai_summary, s.functionoptionid, 42 as latestdatacallid, 'not_started' as status FROM scores s JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid WHERE s.datacallid=$1 ORDER BY s.fismasystemid, fo.functionid, (s.status = 'done') DESC, s.scoreid DESC ON CONFLICT DO NOTHING", sql)
	assert.Equal(t, []any{int32(41)}, args)
}

func TestRolloverSQLTranslatesAcrossVersions(t *testing.T) {
	sql, args := rolloverSQL(41, 42, 7, true)
	assert.Contains(t, sql, versionTranslationJoinSQL("$1"))
	assert.Contains(t, sql, "DISTINCT ON (s.fismasystemid, f.root_functionid)")
	assert.Contains(t, sql, "tfo.functionoptionid, 42 as latestdatacallid")
	assert.Contains(t, sql, "WHERE s.datacallid=$2")
	assert.Equal(t, []any{int32(7), int32(41)}, args)
}

// The preflight decides a re-pin is safe; the write must do exactly what it
// checked, so both are built on one map, and the map on the shared join.
func TestRepinPreflightAndWriteShareTheTranslation(t *testing.T) {
	assert.Contains(t, repinMapSQL, versionTranslationJoinSQL("$2"))
	for name, sql := range map[string]string{
		"preflight": repinPreflightSQL,
		"revisions": repinRevisionsSQL,
		"rewrite":   repinRewriteSQL,
	} {
		assert.Contains(t, sql, "repin_map m", name)
		assert.NotContains(t, sql, "JOIN functions tf", name+" reads the map rather than translating again")
	}
}

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The rollover between calls on one version is the live path every annual
// cycle takes; ztmf-misc#397 changed it only by the trailing ON CONFLICT, and
// ztmf#645 only by the reduced pillar scope on the target call.
func TestRolloverSQLSameVersionIsUnchanged(t *testing.T) {
	sql, args := rolloverSQL(41, 42, 1, false)
	assert.Equal(t, "INSERT INTO scores (fismasystemid,datecalculated,notes,notes_is_ai_summary,functionoptionid,datacallid,status) SELECT DISTINCT ON (s.fismasystemid, fo.functionid) s.fismasystemid, s.datecalculated, s.notes, s.notes_is_ai_summary, s.functionoptionid, 42 as latestdatacallid, 'not_started' as status FROM scores s JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid WHERE s.datacallid=$1 AND "+rolloverScopeSQL("fo.functionid", "$2")+" ORDER BY s.fismasystemid, fo.functionid, (s.status = 'done') DESC, s.scoreid DESC ON CONFLICT DO NOTHING", sql)
	assert.Equal(t, []any{int32(41), int32(42)}, args)
}

func TestRolloverSQLTranslatesAcrossVersions(t *testing.T) {
	sql, args := rolloverSQL(41, 42, 7, true)
	assert.Contains(t, sql, versionTranslationJoinSQL("$1"))
	assert.Contains(t, sql, "DISTINCT ON (s.fismasystemid, f.root_functionid)")
	assert.Contains(t, sql, "tfo.functionoptionid, 42 as latestdatacallid")
	assert.Contains(t, sql, "WHERE s.datacallid=$2")
	assert.Contains(t, sql, rolloverScopeSQL("tf.functionid", "$3"),
		"across versions the scope judges the target function, the row that lands")
	assert.NotContains(t, sql, rolloverScopeSQL("fo.functionid", "$3"))
	assert.Equal(t, []any{int32(7), int32(41), int32(42)}, args)
}

// The rollover's scope must not drop a row through a join: an unmapped or
// NULL-key environment, or a function with no question, has to fall through
// to NOT EXISTS as NULL and keep the row (ztmf#645).
func TestRolloverScopeSQLResolvesThroughScalarSubqueries(t *testing.T) {
	sql := rolloverScopeSQL("fo.functionid", "$2")
	assert.True(t, strings.HasPrefix(sql, "NOT EXISTS ("), "the shared NULL-safe predicate, not a variant")
	assert.Contains(t, sql, "rps.scoring_key = (SELECT dcescope.scoring_key FROM datacenterenvironments dcescope WHERE dcescope.datacenterenvironment = fs.datacenterenvironment)")
	assert.Contains(t, sql, "rp.pillar = (SELECT pscope.pillar FROM functions fscope JOIN questions qscope ON qscope.questionid = fscope.questionid JOIN pillars pscope ON pscope.pillarid = qscope.pillarid WHERE fscope.functionid = fo.functionid)")
	assert.Contains(t, sql, "tdc.datacallid = $2")

	full, _ := rolloverSQL(41, 42, 1, false)
	from := full[strings.Index(full, " FROM scores s "):strings.Index(full, " WHERE ")]
	assert.NotContains(t, from, "datacenterenvironments", "no join on the copy's FROM can drop an answer")
	assert.NotContains(t, from, "questions")
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

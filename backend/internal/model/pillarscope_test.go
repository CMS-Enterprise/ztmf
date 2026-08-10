package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaaSPillarScopeSQL_Shape(t *testing.T) {
	sql := saasPillarScopeSQL("dce.scoring_key", "p.pillar", "$3")

	// A nullable scoring key must not make the whole predicate NULL: a top-level
	// WHERE reads NULL as false and would drop an excluded-pillar row for a
	// non-SaaS system (DECOMMISSIONED maps to scoring_key NULL).
	assert.Contains(t, sql, "COALESCE(dce.scoring_key, '') = 'SaaS'",
		"the scoring key is nullable, so the comparison must be NULL-safe")

	// In scope when named FY26 or dated after every FY26 cycle - never "at or
	// after the earliest FY26 deadline", which a mis-dated FY26 call would use to
	// drag a closed cycle into scope.
	assert.Contains(t, sql, "SELECT MAX(fdc.deadline) FROM datacalls fdc",
		"the cycle gate anchors on the LAST FY26 deadline")
	assert.NotContains(t, sql, "MIN(fdc.deadline)",
		"an earliest-deadline anchor would restate closed cycles")
	assert.Contains(t, sql, "tdc.deadline >", "later cycles are in scope by deadline")

	// Names are matched the way matchesRolloverHardcodeTarget does.
	assert.Equal(t, 2, strings.Count(sql, "UPPER(TRIM(tdc.datacall)) LIKE 'FY2026%'")+
		strings.Count(sql, "UPPER(TRIM(fdc.datacall)) LIKE 'FY2026%'"),
		"both the target and the anchor must trim and upper the name")

	// EXISTS, so a database with no FY26 cycle yields false rather than NULL.
	assert.Contains(t, sql, "EXISTS (")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(sql), "NOT ("),
		"callers append this after AND, so it must be a single parenthesized term")
}

func TestSaaSPillarScopeSQL_UsesCallerExpressions(t *testing.T) {
	sql := saasPillarScopeSQL("(SELECT x FROM y)", "pillars.pillar", "?")

	assert.Contains(t, sql, "COALESCE((SELECT x FROM y), '') = 'SaaS'")
	assert.Contains(t, sql, "pillars.pillar IN ('Devices', 'Applications')")
	assert.Contains(t, sql, "tdc.datacallid = ?")
	assert.NotContains(t, sql, "dce.scoring_key",
		"no alias may be hardcoded; the export has no dce in scope")
}

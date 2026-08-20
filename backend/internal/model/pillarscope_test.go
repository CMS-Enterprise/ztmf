package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReducedPillarScopeSQL_Shape(t *testing.T) {
	sql := reducedPillarScopeSQL("dce.scoring_key", "p.pillar", "$3")

	// NOT EXISTS can never evaluate to NULL, which is what protects the
	// nullable scoring key (DECOMMISSIONED maps to NULL): a top-level WHERE
	// reads NULL as false and would drop an excluded-pillar row for a non-SaaS
	// system. The three-valued behavior itself is pinned in Postgres by
	// TestReducedPillarScopeNullSafetyIntegration.
	assert.True(t, strings.HasPrefix(strings.TrimSpace(sql), "NOT EXISTS ("),
		"callers append this after AND, so it must be a single never-NULL parenthesized term")

	// In scope at or after the seeded anchor's deadline. >= includes the anchor
	// cycle itself.
	assert.Contains(t, sql, "tdc.deadline >= eff.deadline",
		"the anchor cycle and everything after it are in scope")

	// The rule is data (ztmf#545): no cycle names, environments, or pillars may
	// be compiled in.
	assert.Contains(t, sql, "FROM reducedpillarscopes")
	assert.NotContains(t, sql, "LIKE", "name matching came out with the interim predicate")
	assert.NotContains(t, sql, "FY26", "no cycle name may be hardcoded; the anchor is a seeded row")
	assert.NotContains(t, sql, "'SaaS'", "no environment may be hardcoded; the rule rows carry it")
	assert.NotContains(t, sql, "'Devices'", "no pillar may be hardcoded; the rule rows carry it")
}

func TestReducedPillarScopeSQL_UsesCallerExpressions(t *testing.T) {
	sql := reducedPillarScopeSQL("(SELECT x FROM y)", "pillars.pillar", "?")

	assert.Contains(t, sql, "rps.scoring_key = (SELECT x FROM y)")
	assert.Contains(t, sql, "rp.pillar = pillars.pillar")
	assert.Contains(t, sql, "tdc.datacallid = ?")
	assert.NotContains(t, sql, "dce.scoring_key",
		"no alias may be hardcoded; the export has no dce in scope")
}

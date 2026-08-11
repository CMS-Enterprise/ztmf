package model

import (
	"fmt"
	"strings"
)

// reducedPillarScopeCyclePrefixes identifies the cycle the reduced scope starts
// at. Owned here rather than borrowed from rolloverHardcodeTargetPrefixes: that
// slice sits inside the TEMPORARY HARD-CODE block ztmf#502 is queued to delete,
// and depending on it from outside would have blocked that removal.
//
// INTERIM, and not a pattern to copy: matching a cycle by name is a stand-in for
// data. ztmf#545 replaces this with a seeded rule carrying the effective data
// call, at which point this variable and the name matching below both come out.
var reducedPillarScopeCyclePrefixes = []string{"FY2026", "FY26"}

// saasPillarScopeSQL excludes the Devices and Applications pillars for SaaS
// systems from the FY26 cycle onward (ztmf-misc#289). Shared by the progress
// counts and the export so they cannot disagree.
//
// COALESCE on the scoring key is load-bearing: it is nullable (DECOMMISSIONED
// maps to NULL), and NOT(NULL AND ...) is NULL, which a top-level WHERE treats as
// false - silently dropping an excluded-pillar row for a non-SaaS system. Every
// condition must resolve to true or false, never NULL, because callers put this in
// both a WHERE and a JOIN ... ON.
//
// A cycle is in scope when it is named FY26 or its deadline falls after every FY26
// cycle. Deliberately not "deadline >= the earliest FY26 deadline": an FY26 call
// mis-dated before a closed cycle would drag that closed cycle into scope and
// restate it. Names are trimmed and uppercased before matching.
//
// Not scoped by OpDiv yet: CMS is expected to keep all 40, but the frontend filter
// is OpDiv-blind, so exempting CMS here alone would leave those systems unable to
// answer the 15 questions a 40 denominator demands. When confirmed, add alongside
// the frontend change:
//
//	AND EXISTS (SELECT 1 FROM opdivs o WHERE o.opdiv_id = <system>.opdiv_id AND UPPER(o.code) <> 'CMS')
//
// Arguments are SQL expressions valid in the caller's scope. dataCallExpr is the
// placeholder for the call being read ("$3" hand-built, "?" via squirrel).
func saasPillarScopeSQL(scoringKeyExpr, pillarExpr, dataCallExpr string) string {
	prefixCond := func(alias string) string {
		conds := make([]string, len(reducedPillarScopeCyclePrefixes))
		for i, prefix := range reducedPillarScopeCyclePrefixes {
			conds[i] = fmt.Sprintf("UPPER(TRIM(%s.datacall)) LIKE '%s%%'", alias, prefix)
		}
		return strings.Join(conds, " OR ")
	}

	return fmt.Sprintf(`NOT (
          COALESCE(%s, '') = 'SaaS'
          AND %s IN ('Devices', 'Applications')
          AND EXISTS (
              SELECT 1 FROM datacalls tdc
              WHERE tdc.datacallid = %s
                AND (
                    (%s)
                    OR tdc.deadline > (
                        SELECT MAX(fdc.deadline) FROM datacalls fdc WHERE %s
                    )
                )
          )
      )`,
		scoringKeyExpr,
		pillarExpr,
		dataCallExpr,
		prefixCond("tdc"),
		prefixCond("fdc"),
	)
}

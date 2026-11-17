package model

import (
	"fmt"
	"strings"
)

// reducedPillarScopeSQL excludes the pillars a seeded rule removes from a
// system's question set (ztmf#545, ztmf-misc#289). A reducedpillarscopes row
// (scoring_key, pillarid, effective_datacallid) puts a pillar out of scope for
// systems scored under that key, on every data call whose deadline is on or
// after the anchor call's. Shared by the progress counts, the export, the
// scoring aggregate, the questionnaire, the data call rollover
// (rolloverScopeSQL, ztmf#645) and Compare Data Calls (scoreCycleCTE, through
// reducedPillarScopeAnySQL) so they cannot disagree.
//
// NOT EXISTS carries the NULL-safety the callers rely on: it never evaluates
// to NULL, and a NULL scoring key (DECOMMISSIONED maps to NULL) fails the
// equality inside the subquery, so the row is kept. Every caller puts this in
// a WHERE or a JOIN ... ON, both of which read NULL as "drop the row".
//
// The boundary is the seeded anchor's deadline, nothing else: a cycle created
// later with a misleading name or date cannot move it, which is the hazard the
// interim name-prefix predicate this replaces had to anchor on MAX(deadline)
// to avoid. An unknown data-call id joins to nothing and falls open to the
// full question set, matching how other endpoints treat unresolvable filters.
//
// Arguments are SQL expressions valid in the caller's scope. dataCallExpr is
// the placeholder or column for the call being read ("$3" hand-built, "?" via
// squirrel, "sp.datacallid" inside a CTE).
func reducedPillarScopeSQL(scoringKeyExpr, pillarExpr, dataCallExpr string) string {
	return reducedPillarScopeForSQL(scoringKeyExpr, pillarExpr, "= "+dataCallExpr)
}

// reducedPillarScopeAnySQL is reducedPillarScopeSQL across several calls: the
// pillar is kept only if it is in scope in every one of them, in one NOT
// EXISTS rather than one per call. Compare Data Calls uses it to judge a row
// against both cycles. The rule is monotone in deadline, so this equals
// judging against the later call alone.
// It takes at least one call so it can never build an empty IN list.
func reducedPillarScopeAnySQL(scoringKeyExpr, pillarExpr, firstCall string, otherCalls ...string) string {
	calls := append([]string{firstCall}, otherCalls...)
	return reducedPillarScopeForSQL(scoringKeyExpr, pillarExpr, "IN ("+strings.Join(calls, ", ")+")")
}

// reducedPillarScopeForSQL is the predicate both share; callMatch completes
// "tdc.datacallid ".
func reducedPillarScopeForSQL(scoringKeyExpr, pillarExpr, callMatch string) string {
	return fmt.Sprintf(`NOT EXISTS (
          SELECT 1
          FROM reducedpillarscopes rps
          INNER JOIN pillars rp ON rp.pillarid = rps.pillarid
          INNER JOIN datacalls eff ON eff.datacallid = rps.effective_datacallid
          INNER JOIN datacalls tdc ON tdc.datacallid %s
          WHERE rps.scoring_key = %s
            AND rp.pillar = %s
            AND tdc.deadline >= eff.deadline
      )`,
		callMatch,
		scoringKeyExpr,
		pillarExpr,
	)
}

// answerScopeSQL is reducedPillarScopeSQL for a stored answer, for a caller
// that reads scores rather than walks the catalog and has no join of its own
// to hang the lookups on: the rollover. environmentExpr is the system's raw datacenterenvironment and
// functionExpr the function the answer sits on, both valid in the caller's
// scope.
//
// Both lookups are scalar subqueries rather than joins on the caller's FROM,
// so they cannot drop a row: an environment that is unmapped or maps to a NULL
// scoring key, or a function with no question, yields NULL, and the NOT EXISTS
// keeps the row. Each subquery walks primary keys, so it returns at most one
// row.
func answerScopeSQL(environmentExpr, functionExpr, dataCallExpr string) string {
	return reducedPillarScopeSQL(
		"(SELECT dcescope.scoring_key FROM datacenterenvironments dcescope WHERE dcescope.datacenterenvironment = "+environmentExpr+")",
		"(SELECT pscope.pillar FROM functions fscope"+
			" JOIN questions qscope ON qscope.questionid = fscope.questionid"+
			" JOIN pillars pscope ON pscope.pillarid = qscope.pillarid"+
			" WHERE fscope.functionid = "+functionExpr+")",
		dataCallExpr,
	)
}

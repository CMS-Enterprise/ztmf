package model

import (
	"fmt"
)

// reducedPillarScopeSQL excludes the pillars a seeded rule removes from a
// system's question set (ztmf#545, ztmf-misc#289). A reducedpillarscopes row
// (scoring_key, pillarid, effective_datacallid) puts a pillar out of scope for
// systems scored under that key, on every data call whose deadline is on or
// after the anchor call's. Shared by the progress counts, the export, the
// scoring aggregate and the questionnaire so they cannot disagree.
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
	return fmt.Sprintf(`NOT EXISTS (
          SELECT 1
          FROM reducedpillarscopes rps
          INNER JOIN pillars rp ON rp.pillarid = rps.pillarid
          INNER JOIN datacalls eff ON eff.datacallid = rps.effective_datacallid
          INNER JOIN datacalls tdc ON tdc.datacallid = %s
          WHERE rps.scoring_key = %s
            AND rp.pillar = %s
            AND tdc.deadline >= eff.deadline
      )`,
		dataCallExpr,
		scoringKeyExpr,
		pillarExpr,
	)
}

package model

import (
	"fmt"
	"strings"
)

// saasPillarScopeSQL excludes the Devices and Applications pillars for SaaS
// systems from the FY26 cycle onward (ztmf-misc#289). Shared by the progress
// counts and the export so they cannot disagree.
//
// Anchored on deadline, not datacallid: ids are not chronological, so a backfill
// loaded later must still read as history. EXISTS rather than a bare comparison
// because callers place this in a JOIN ... ON, where NULL drops the row - a
// database with no FY26 call has to fall back to the full 40.
//
// Not scoped by OpDiv yet: CMS is expected to keep all 40, but the frontend
// filter is OpDiv-blind, so exempting CMS here alone would leave those systems
// unable to answer the 15 questions a 40 denominator demands. When confirmed, add
// alongside the frontend change:
//
//	AND EXISTS (SELECT 1 FROM opdivs o WHERE o.opdiv_id = <system>.opdiv_id AND UPPER(o.code) <> rolloverHardcodeCMSOpDiv)
//
// Arguments are SQL expressions valid in the caller's scope. dataCallExpr is the
// placeholder for the call being read ("$3" hand-built, "?" via squirrel).
func saasPillarScopeSQL(scoringKeyExpr, pillarExpr, dataCallExpr string) string {
	prefixConds := make([]string, len(rolloverHardcodeTargetPrefixes))
	for i, prefix := range rolloverHardcodeTargetPrefixes {
		prefixConds[i] = fmt.Sprintf("UPPER(fdc.datacall) LIKE '%s%%'", prefix)
	}

	return fmt.Sprintf(`NOT (
          %s = 'SaaS'
          AND %s IN ('Devices', 'Applications')
          AND EXISTS (
              SELECT 1 FROM datacalls tdc
              WHERE tdc.datacallid = %s
                AND tdc.deadline >= (
                    SELECT MIN(fdc.deadline) FROM datacalls fdc WHERE %s
                )
          )
      )`, scoringKeyExpr, pillarExpr, dataCallExpr, strings.Join(prefixConds, " OR "))
}

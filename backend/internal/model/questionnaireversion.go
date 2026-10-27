package model

import (
	"fmt"
)

// questionnaireVersionSQL restricts a functions row to the catalog version the
// data call is pinned to (ztmf-misc#395). Shared by every query that
// enumerates applicable functions, and by the score write path, so they cannot
// disagree about which catalog a cycle answers.
//
// Its polarity is the opposite of reducedPillarScopeSQL's on purpose. The
// pillar scope is subtractive, so an unresolvable data call falls open to the
// full question set. A version is a selector: falling open would mean every
// version's catalog at once, multiplying the questionnaire and the scoring
// denominator into a wrong number rather than a wide list. So this is
// "= (scalar subquery)": an unknown data call yields NULL, which WHERE and
// JOIN ... ON both read as "drop the row". An empty questionnaire is loud; a
// doubled denominator is silent.
//
// Pinning functions.versionid alone is sufficient: the composite FKs from
// migration 0068 mean the questions and functionoptions rows a function joins
// to are in the same version.
//
// The version is one of three independent axes. functions.datacenterenvironment
// picks the edition, and reducedpillarscopes (unversioned, effective-dated on
// its own) picks the pillars in scope. Keep them separate.
//
// Arguments are SQL expressions valid in the caller's scope: versionExpr is
// the functions row's versionid column, dataCallExpr the placeholder or column
// for the call being read ("$3" hand-built, "?" via squirrel, "sp.datacallid"
// inside a CTE).
func questionnaireVersionSQL(versionExpr, dataCallExpr string) string {
	return fmt.Sprintf(`%s = (SELECT qvdc.questionnaire_versionid FROM datacalls qvdc WHERE qvdc.datacallid = %s)`,
		versionExpr,
		dataCallExpr,
	)
}

// pinnedVersionJoinSQL is questionnaireVersionSQL for a query that already
// joins the data call: the same rule as a plain join condition, so the planner
// can filter functions by it instead of running a lookup per row. dataCallAlias
// must be INNER JOINed, so it never yields the NULL the subquery form guards.
func pinnedVersionJoinSQL(versionExpr, dataCallAlias string) string {
	return fmt.Sprintf(`%s = %s.questionnaire_versionid`, versionExpr, dataCallAlias)
}

// currentQuestionnaireVersionSQL is questionnaireVersionSQL for a read that
// names no data call: the newest published version. It fails closed the same
// way, since no published version yields NULL.
func currentQuestionnaireVersionSQL(versionExpr string) string {
	return fmt.Sprintf(`%s = (SELECT MAX(qvcur.versionid) FROM questionnaire_versions qvcur WHERE qvcur.state = 'published')`,
		versionExpr,
	)
}

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuestionnaireVersionSQL_Text(t *testing.T) {
	cases := map[string]struct{ version, dataCall, want string }{
		"squirrel placeholder": {"functions.versionid", "?",
			"functions.versionid = (SELECT qvdc.questionnaire_versionid FROM datacalls qvdc WHERE qvdc.datacallid = ?)"},
		"numbered placeholder": {"f.versionid", "$3",
			"f.versionid = (SELECT qvdc.questionnaire_versionid FROM datacalls qvdc WHERE qvdc.datacallid = $3)"},
		"column expression": {"f.versionid", "sp.datacallid",
			"f.versionid = (SELECT qvdc.questionnaire_versionid FROM datacalls qvdc WHERE qvdc.datacallid = sp.datacallid)"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, questionnaireVersionSQL(tc.version, tc.dataCall))
		})
	}
}

// Fail closed: "=" against a scalar subquery is NULL for an unknown data call
// or a missing published version, which drops the row. A NOT EXISTS shape like
// the pillar scope's would fall open to every version at once.
func TestQuestionnaireVersionSQL_FailsClosed(t *testing.T) {
	for _, sql := range []string{
		questionnaireVersionSQL("f.versionid", "$1"),
		currentQuestionnaireVersionSQL("versionid"),
	} {
		assert.Contains(t, sql, " = (SELECT ")
		assert.NotContains(t, sql, "EXISTS")
		assert.NotContains(t, sql, "COALESCE")
	}
	assert.Equal(t,
		"versionid = (SELECT MAX(qvcur.versionid) FROM questionnaire_versions qvcur WHERE qvcur.state = 'published')",
		currentQuestionnaireVersionSQL("versionid"))
}

func TestBuildScoreProgressSQL_VersionInBothCTEs(t *testing.T) {
	dc := int32(5)
	sql, args := buildScoreProgressSQL(FindScoreProgressInput{DataCallID: &dc})
	// Same reason as the pillar scope: in expected alone, answered and updated
	// could count functions outside the version and exceed the denominator.
	fragment := questionnaireVersionSQL("f.versionid", "$1")
	assert.Equal(t, 2, strings.Count(sql, fragment))
	assert.Equal(t, []any{dc}, args)
}

func TestBuildPillarScoresSQL_VersionPerRow(t *testing.T) {
	sql, _ := buildPillarScoresSQL(FindScoresInput{})
	assert.Equal(t, 1, strings.Count(sql, "f.versionid = dc.questionnaire_versionid"),
		"anchored on each pair's own data call, so a closed cycle rescores against its own catalog")
	assert.Contains(t, sql, "INNER JOIN datacalls dc    ON dc.datacallid    = sp.datacallid",
		"the join form is only fail-closed while dc is an inner join on the pair's data call")
	assert.NotContains(t, sql, "drift is intentionally not handled")
}

func TestBuildScoreDiffSQL_JoinsOnLineage(t *testing.T) {
	from, to := int32(1), int32(2)
	sql, _ := buildScoreDiffSQL(FindScoreDiffInput{FromDataCallID: &from, ToDataCallID: &to})
	assert.Contains(t, sql, "f.root_functionid = t.root_functionid")
	assert.NotContains(t, sql, "AND f.functionid = t.functionid", "the cycles may pin different versions")
	assert.Contains(t, sql, "LEFT JOIN functions fn ON fn.functionid = COALESCE(t.functionid, f.functionid)",
		"the label join stays on the primary key so it cannot fan out, and labels come from the cycle under review")
	assert.Contains(t, sql, "ELSE f.option_score IS DISTINCT FROM t.option_score",
		"across versions an unchanged answer has a new option id")
}

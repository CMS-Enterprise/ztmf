package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestFindScoresInput_QueryCannotWiden pins that the scores list endpoints'
// server-owned scope fields are not bindable from the query string.
//
// ListScores and GetScoresAggregate decode the client query onto the input
// before ApplyTier runs, and ApplyTier only overwrites UserID for the tiers that
// need self-scope. For every other tier it returns false and leaves the decoded
// value in place, so the tag is what keeps a client-supplied value out of the
// query in the first place.
func TestFindScoresInput_QueryCannotWiden(t *testing.T) {
	for _, q := range []string{"UserID", "FismaSystemIDs"} {
		t.Run(q, func(t *testing.T) {
			input := model.FindScoresInput{}
			err := decoder.Decode(&input, url.Values{q: {"1"}})

			assert.Error(t, err,
				"a server-owned scope field must not be bindable from the query string")
		})
	}
}

// TestFindScoresInput_LegitimateQueryParamsStillDecode is the companion positive
// case: tagging the server-owned fields must not break the parameters callers
// are supposed to be able to send.
func TestFindScoresInput_LegitimateQueryParamsStillDecode(t *testing.T) {
	input := model.FindScoresInput{}
	err := decoder.Decode(&input, url.Values{
		"fismasystemid":   {"7"},
		"datacallid":      {"3"},
		"include_pillars": {"true"},
	})

	assert.NoError(t, err)
	if assert.NotNil(t, input.FismaSystemID) {
		assert.Equal(t, int32(7), *input.FismaSystemID)
	}
	if assert.NotNil(t, input.DataCallID) {
		assert.Equal(t, int32(3), *input.DataCallID)
	}
	if assert.NotNil(t, input.IncludePillars) {
		assert.True(t, *input.IncludePillars)
	}
}

// TestListScores_ScopeFieldsRejectedFromQuery pins the same boundary at the
// handler: a request naming a server-owned scope field returns 400 for the
// unknown key rather than 200 with redirected scope. Decode failing short
// circuits before the query runs, so this needs no database.
func TestListScores_ScopeFieldsRejectedFromQuery(t *testing.T) {
	for _, q := range []string{"UserID=victim-uuid", "userid=victim-uuid", "FismaSystemIDs=1"} {
		t.Run(q, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/scores?"+q, nil)
			r = withUser(r, issoUser)
			w := httptest.NewRecorder()

			ListScores(w, r)

			assert.Equal(t, http.StatusBadRequest, w.Code,
				"a server-owned scope field in the query must 400, not redirect scope")
		})
	}
}

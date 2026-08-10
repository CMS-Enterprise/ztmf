package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

func int32PtrExport(v int32) *int32 { return &v }

// TestScopeFindAnswersInput pins the role matrix the data-call export applies to
// its query input. This is the security boundary for GET /datacalls/{id}/export:
// the SQL is tested separately, so what matters here is that each tier populates
// exactly the scope fields FindAnswers keys on, and that no tier is dropped into
// the wrong branch (ztmf-misc#267).
func TestScopeFindAnswersInput(t *testing.T) {
	t.Run("OwnerUnrestricted", func(t *testing.T) {
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(adminUser, &input)

		assert.False(t, input.RestrictToOpDivIDs, "OWNER must not be OpDiv-restricted")
		assert.Empty(t, input.OpDivIDs)
		assert.Nil(t, input.UserID, "OWNER must not be limited to assigned systems")
	})

	t.Run("HHSReadonlyAdminUnrestricted", func(t *testing.T) {
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(readonlyAdmin, &input)

		assert.False(t, input.RestrictToOpDivIDs, "HHS_READONLY_ADMIN has unscoped read")
		assert.Empty(t, input.OpDivIDs)
		assert.Nil(t, input.UserID)
	})

	t.Run("OpDivAdminScopedToGrants", func(t *testing.T) {
		opdivAdmin := &model.User{
			UserID:           "44444444-4444-4444-4444-444444444444",
			Role:             "OPDIV_ADMIN",
			AssignedOpDivIDs: []*int32{int32PtrExport(7), int32PtrExport(9)},
		}
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(opdivAdmin, &input)

		assert.True(t, input.RestrictToOpDivIDs, "OPDIV_ADMIN must be OpDiv-restricted (ztmf-misc#267)")
		assert.Equal(t, []int32{7, 9}, input.OpDivIDs)
		assert.Nil(t, input.UserID, "OpDiv tier scopes by OpDiv, not by assignment")
	})

	t.Run("OpDivReadonlyAdminScopedLikeOpDivAdmin", func(t *testing.T) {
		opdivReadonly := &model.User{
			UserID:           "66666666-6666-6666-6666-666666666666",
			Role:             "OPDIV_READONLY_ADMIN",
			AssignedOpDivIDs: []*int32{int32PtrExport(3)},
		}
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(opdivReadonly, &input)

		assert.True(t, input.RestrictToOpDivIDs)
		assert.Equal(t, []int32{3}, input.OpDivIDs)
		assert.Nil(t, input.UserID)
	})

	t.Run("OpDivAdminWithNoGrantsFailsClosed", func(t *testing.T) {
		opdivAdmin := &model.User{
			UserID: "55555555-5555-5555-5555-555555555555",
			Role:   "OPDIV_ADMIN",
		}
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(opdivAdmin, &input)

		assert.True(t, input.RestrictToOpDivIDs)
		assert.Empty(t, input.OpDivIDs, "no grants must leave the id list empty so the query fails closed")
	})

	t.Run("ISSOScopedToAssignedSystems", func(t *testing.T) {
		input := model.FindAnswersInput{}
		scopeFindAnswersInput(issoUser, &input)

		assert.False(t, input.RestrictToOpDivIDs)
		if assert.NotNil(t, input.UserID, "ISSO must be limited to their assigned systems") {
			assert.Equal(t, issoUser.UserID, *input.UserID)
		}
	})
}

// TestScopeFindAnswersInput_QueryCannotWiden is the negative case: an ISSO sends
// a hostile query, the handler decodes it and then scopes, and the caller's own
// identity must win. This pins the decode-then-scope ordering that ztmf-misc#268
// was about, at the unit level, without a database or the full handler.
func TestScopeFindAnswersInput_QueryCannotWiden(t *testing.T) {
	// Mimic the handler: decode the client query onto the input, THEN scope.
	input := model.FindAnswersInput{}
	err := decoder.Decode(&input, url.Values{"UserID": {"victim-uuid"}})
	// UserID is schema:"-", so it is an unknown key and decode rejects it. Even
	// if that changed, the scope step below would still overwrite it.
	assert.Error(t, err, "a server-owned field must not be bindable from the query string")

	input = model.FindAnswersInput{UserID: int32StrPtr("victim-uuid")}
	scopeFindAnswersInput(issoUser, &input)
	if assert.NotNil(t, input.UserID) {
		assert.Equal(t, issoUser.UserID, *input.UserID,
			"scope runs after decode, so the caller's own id must win over any client value")
	}
}

func int32StrPtr(s string) *string { return &s }

// TestGetDatacallExport_ScopeFieldsRejectedFromQuery pins that the export's
// server-owned scope fields cannot be set from the query string: a request that
// tries returns 400 (unknown key) rather than 200 with redirected scope. fsids
// remains the one legitimate query parameter.
func TestGetDatacallExport_ScopeFieldsRejectedFromQuery(t *testing.T) {
	for _, q := range []string{"UserID=victim-uuid", "userid=victim-uuid", "RestrictToOpDivIDs=false", "OpDivIDs=1"} {
		t.Run(q, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/datacalls/1/export?"+q, nil)
			r = withUser(r, issoUser)
			w := httptest.NewRecorder()

			GetDatacallExport(w, r)

			assert.Equal(t, http.StatusBadRequest, w.Code,
				"a server-owned scope field in the query must 400, not redirect scope")
		})
	}
}

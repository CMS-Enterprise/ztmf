package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

// SaveOpDiv is OWNER-only: the OpDiv list is the tenant boundary, so even
// HHS_ADMIN and OPDIV_ADMIN cannot create or change one. The IsOwner gate runs
// before any body parse or DB call, so the forbidden cases need no database.
func TestSaveOpDiv_OwnerOnly(t *testing.T) {
	forbidden := []*model.User{
		{Role: "HHS_ADMIN"},
		{Role: "HHS_READONLY_ADMIN"},
		{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{ptr32(2)}},
		{Role: "OPDIV_READONLY_ADMIN"},
		{Role: "ISSO"},
	}
	for _, u := range forbidden {
		t.Run("POST forbidden for "+u.Role, func(t *testing.T) {
			body := jsonBody(t, map[string]any{"code": "TEST", "name": "Test OpDiv"})
			r := withUser(httptest.NewRequest("POST", "/api/v1/opdivs", body), u)
			w := httptest.NewRecorder()
			SaveOpDiv(w, r)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func ptr32(i int32) *int32 { return &i }

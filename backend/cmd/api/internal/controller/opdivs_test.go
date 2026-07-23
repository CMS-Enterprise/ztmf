package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
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

// SetOpDivSystemDelegateEnabled is settable by Owner and HHS admin only (#467
// decision 7). Unlike SaveOpDiv it is NOT OWNER-only - HHS_ADMIN must pass - but
// an OPDIV_ADMIN (though IsAdmin) and every scoped/read-only tier must be
// rejected. The CanWriteHHSWide gate runs before any DB call.
func TestSetOpDivSystemDelegateEnabled_HHSWideOnly(t *testing.T) {
	body := func() *httptest.ResponseRecorder { return httptest.NewRecorder() }

	forbidden := []*model.User{
		{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{ptr32(2)}},
		{Role: "OPDIV_READONLY_ADMIN"},
		{Role: "HHS_READONLY_ADMIN"},
		{Role: "ISSO"},
		{Role: "SYSTEM_DELEGATE"},
	}
	for _, u := range forbidden {
		t.Run("forbidden for "+u.Role, func(t *testing.T) {
			r := withUser(httptest.NewRequest("PUT", "/api/v1/opdivs/2/system-delegate-enabled", jsonBody(t, map[string]any{"enabled": true})), u)
			r = mux.SetURLVars(r, map[string]string{"opdiv_id": "2"})
			w := body()
			SetOpDivSystemDelegateEnabled(w, r)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}

	allowed := []*model.User{
		{Role: "OWNER"},
		{Role: "HHS_ADMIN"},
	}
	for _, u := range allowed {
		t.Run("gate passes for "+u.Role, func(t *testing.T) {
			r := withUser(httptest.NewRequest("PUT", "/api/v1/opdivs/2/system-delegate-enabled", jsonBody(t, map[string]any{"enabled": true})), u)
			r = mux.SetURLVars(r, map[string]string{"opdiv_id": "2"})
			w := body()
			SetOpDivSystemDelegateEnabled(w, r)
			// Passes the role gate; without a DB it fails downstream, but must not be 403.
			assert.NotEqual(t, http.StatusForbidden, w.Code)
		})
	}
}

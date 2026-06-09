package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

// opdivPtr is a local int32-pointer helper for building AssignedOpDivIDs in the
// OpDiv-scoped tier fixtures below.
func opdivPtr(v int32) *int32 { return &v }

var (
	opdivAdmin = &model.User{
		UserID:           "88888888-8888-4888-8888-888888888888",
		Email:            "opdiv.admin@test.com",
		Role:             "OPDIV_ADMIN",
		AssignedOpDivIDs: []*int32{opdivPtr(1)},
	}
	opdivReadonly = &model.User{
		UserID:           "99999999-9999-4999-8999-999999999999",
		Email:            "opdiv.readonly@test.com",
		Role:             "OPDIV_READONLY_ADMIN",
		AssignedOpDivIDs: []*int32{opdivPtr(1)},
	}
)

// These gates all return before any database access, so the forbidden cases are
// pure no-DB unit tests. The allowed paths are intentionally not exercised here
// because they would fall through to a DB query (and can hang without DB env);
// they are covered by the isolated Emberfall E2E matrix instead.

// --- GetEvents: restricted to unscoped admins (no opdiv_id to scope the log) ---

func TestGetEvents_OpDivAdminForbidden(t *testing.T) {
	r := withUser(httptest.NewRequest("GET", "/api/v1/events", nil), opdivAdmin)
	w := httptest.NewRecorder()
	GetEvents(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetEvents_OpDivReadonlyForbidden(t *testing.T) {
	r := withUser(httptest.NewRequest("GET", "/api/v1/events", nil), opdivReadonly)
	w := httptest.NewRecorder()
	GetEvents(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveMassEmail: restricted to unscoped WRITE admins (OWNER / HHS_ADMIN) ---

func TestSaveMassEmail_OpDivAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{"subject": "x", "body": "y"})
	r := withUser(httptest.NewRequest("POST", "/api/v1/massemails", body), opdivAdmin)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveMassEmail(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveMassEmail_OpDivReadonlyForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{"subject": "x", "body": "y"})
	r := withUser(httptest.NewRequest("POST", "/api/v1/massemails", body), opdivReadonly)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveMassEmail(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveScore: read-only tiers are blocked before any DB access ---

func TestSaveScore_OpDivReadonlyForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{"fismasystemid": 1002, "functionoptionid": 1, "datacallid": 3})
	r := withUser(httptest.NewRequest("POST", "/api/v1/scores", body), opdivReadonly)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

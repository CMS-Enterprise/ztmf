package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const grantUserID = "11111111-1111-4111-8111-111111111111"

// SetUserOpDivs forbidden paths are reached before any DB call: a non-admin is
// rejected by the tier gate, and an OPDIV_ADMIN requesting an out-of-scope OpDiv
// is rejected by the in-memory scope check.
func TestSetUserOpDivs_Forbidden(t *testing.T) {
	t.Run("non-admin tier is forbidden", func(t *testing.T) {
		r := withUser(httptest.NewRequest("PUT", "/api/v1/users/"+grantUserID+"/opdivs",
			jsonBody(t, map[string]any{"opdiv_ids": []int{3}})), &model.User{Role: "ISSO"})
		r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
		w := httptest.NewRecorder()
		SetUserOpDivs(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("OPDIV_ADMIN cannot request an OpDiv they do not hold", func(t *testing.T) {
		opdiv2 := int32(2)
		actor := &model.User{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{&opdiv2}}
		r := withUser(httptest.NewRequest("PUT", "/api/v1/users/"+grantUserID+"/opdivs",
			jsonBody(t, map[string]any{"opdiv_ids": []int{99}})), actor)
		r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
		w := httptest.NewRecorder()
		SetUserOpDivs(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// SetUserOpDivs returns 400 for a malformed request body before touching the DB.
func TestSetUserOpDivs_MalformedBody(t *testing.T) {
	r := withUser(httptest.NewRequest("PUT", "/api/v1/users/"+grantUserID+"/opdivs",
		strings.NewReader("not-json")), &model.User{Role: "HHS_ADMIN"})
	r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
	w := httptest.NewRecorder()
	SetUserOpDivs(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// An OPDIV_ADMIN requesting only their own OpDivs clears the scope gate. The
// handler proceeds to the tier-ceiling DB lookup (no test DB) and returns 500
// — crucially NOT 403, which would mean the gate incorrectly blocked them.
func TestSetUserOpDivs_OpDivAdminInScopePassesGate(t *testing.T) {
	opdiv5 := int32(5)
	actor := &model.User{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{&opdiv5}}
	r := withUser(httptest.NewRequest("PUT", "/api/v1/users/"+grantUserID+"/opdivs",
		jsonBody(t, map[string]any{"opdiv_ids": []int{5}})), actor)
	r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
	w := httptest.NewRecorder()
	SetUserOpDivs(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// CreateUserOpDiv's forbidden paths are reached before any DB call: a non-admin
// is rejected by the tier gate, and an OPDIV_ADMIN granting an OpDiv they do not
// hold is rejected by the in-memory scope check.
func TestCreateUserOpDiv_Forbidden(t *testing.T) {
	t.Run("non-admin tier is forbidden", func(t *testing.T) {
		r := withUser(httptest.NewRequest("POST", "/api/v1/users/"+grantUserID+"/assignedopdivs",
			jsonBody(t, map[string]any{"opdiv_id": 3})), &model.User{Role: "ISSO"})
		r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
		w := httptest.NewRecorder()
		CreateUserOpDiv(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("OPDIV_ADMIN cannot grant an OpDiv they do not hold", func(t *testing.T) {
		opdiv2 := int32(2)
		actor := &model.User{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{&opdiv2}}
		r := withUser(httptest.NewRequest("POST", "/api/v1/users/"+grantUserID+"/assignedopdivs",
			jsonBody(t, map[string]any{"opdiv_id": 99})), actor)
		r = mux.SetURLVars(r, map[string]string{"userid": grantUserID})
		w := httptest.NewRecorder()
		CreateUserOpDiv(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

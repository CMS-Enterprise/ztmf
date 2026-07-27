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

// Injection defense: the add body must accept only email/fullname/access_expires_at,
// so a client can never smuggle role/opdiv/userid to escalate. getJSON sets
// DisallowUnknownFields; this pins that guarantee for the delegate payload.
func TestAddDelegateBody_RejectsUnknownFields(t *testing.T) {
	err := getJSON(strings.NewReader(`{"email":"x@y.z","fullname":"X","role":"OWNER"}`), &addDelegateBody{})
	assert.Error(t, err, "an unknown field (role) must be rejected, not silently ignored")

	err = getJSON(strings.NewReader(`{"email":"x@y.z","fullname":"X"}`), &addDelegateBody{})
	assert.NoError(t, err, "a clean body must parse")
}

// These tests pin the delegate-management authorization gate, which is decided
// in memory before any DB work (guardManageDelegates' pre-check). A denied actor
// must get 404 (not-leak) without ever reaching the database, so these assertions
// hold with no DB. Allowed actors pass the gate and then fail later on the absent
// DB; the happy paths are covered by the integration suite, so here we only assert
// that an allowed actor is NOT gated (not 404/403).

const (
	delegateSystemID   = "1"
	delegateTargetUUID = "66666666-6666-4666-8666-666666666666"
)

func delegateReq(t *testing.T, method, body string, u *model.User, vars map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, "/api/v1/fismasystems/1/delegates", jsonBody(t, map[string]any{"email": "new@empire.test", "fullname": "New Delegate"}))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, "/api/v1/fismasystems/1/delegates", nil)
	}
	r = mux.SetURLVars(r, vars)
	return httptest.NewRecorder(), withUser(r, u)
}

// deniedActors can never manage delegates on a system: a delegate, an ISSM (even
// assigned), an ISSO not assigned to the system, and the read-only tiers.
func deniedActors() map[string]*model.User {
	return map[string]*model.User{
		"delegate assigned": {UserID: "55555555-5555-4555-8555-555555555555", Role: "SYSTEM_DELEGATE", AssignedFismaSystems: []*int32{int32Ptr(1)}},
		"ISSM assigned":     {UserID: "77777777-7777-4777-8777-777777777777", Role: "ISSM", AssignedFismaSystems: []*int32{int32Ptr(1)}},
		"ISSO unassigned":   {UserID: "33333333-3333-4333-8333-333333333333", Role: "ISSO"},
		"readonly admin":    readonlyAdmin,
	}
}

func TestAddSystemDelegate_DeniedActorsNotFound(t *testing.T) {
	for name, u := range deniedActors() {
		t.Run(name, func(t *testing.T) {
			w, r := delegateReq(t, "POST", "body", u, map[string]string{"fismasystemid": delegateSystemID})
			AddSystemDelegate(w, r)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestRemoveSystemDelegate_DeniedActorsNotFound(t *testing.T) {
	for name, u := range deniedActors() {
		t.Run(name, func(t *testing.T) {
			w, r := delegateReq(t, "DELETE", "", u, map[string]string{"fismasystemid": delegateSystemID, "userid": delegateTargetUUID})
			RemoveSystemDelegate(w, r)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestRenewSystemDelegate_DeniedActorsNotFound(t *testing.T) {
	for name, u := range deniedActors() {
		t.Run(name, func(t *testing.T) {
			w, r := delegateReq(t, "PATCH", "", u, map[string]string{"fismasystemid": delegateSystemID, "userid": delegateTargetUUID})
			RenewSystemDelegate(w, r)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

// A delegate assigned to the system must not be able to read the roster (the
// section is hidden from delegates); rejected in memory as 404 before any DB.
func TestListSystemDelegates_DelegateForbidden(t *testing.T) {
	delegate := &model.User{
		UserID:               "55555555-5555-4555-8555-555555555555",
		Role:                 "SYSTEM_DELEGATE",
		AssignedFismaSystems: []*int32{int32Ptr(1)},
	}
	r := httptest.NewRequest("GET", "/api/v1/fismasystems/1/delegates", nil)
	r = mux.SetURLVars(r, map[string]string{"fismasystemid": delegateSystemID})
	w := httptest.NewRecorder()
	ListSystemDelegates(w, withUser(r, delegate))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListDelegateCandidates_DeniedActorsNotFound(t *testing.T) {
	for name, u := range deniedActors() {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/fismasystems/1/delegate-candidates?q=foo", nil)
			r = mux.SetURLVars(r, map[string]string{"fismasystemid": delegateSystemID})
			w := httptest.NewRecorder()
			ListDelegateCandidates(w, withUser(r, u))
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

// An ISSO assigned to the system, and an unscoped admin, both pass the gate; with
// no DB they fail downstream, but must not be gated with 403/404.
func TestAddSystemDelegate_AllowedActorsPassGate(t *testing.T) {
	issoAssigned := &model.User{UserID: "33333333-3333-4333-8333-333333333333", Role: "ISSO", AssignedFismaSystems: []*int32{int32Ptr(1)}}
	for name, u := range map[string]*model.User{"ISSO assigned": issoAssigned, "OWNER": adminUser} {
		t.Run(name, func(t *testing.T) {
			w, r := delegateReq(t, "POST", "body", u, map[string]string{"fismasystemid": delegateSystemID})
			AddSystemDelegate(w, r)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "gate must pass for %s", name)
			assert.NotEqual(t, http.StatusForbidden, w.Code, "gate must pass for %s", name)
		})
	}
}

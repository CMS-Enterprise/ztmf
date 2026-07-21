package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// withUser creates an HTTP request with the given user injected into context,
// matching how the auth middleware sets up requests.
func withUser(r *http.Request, user *model.User) *http.Request {
	ctx := model.UserToContext(r.Context(), user)
	return r.WithContext(ctx)
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

var (
	adminUser = &model.User{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "admin@test.com",
		Role:   "OWNER",
	}
	readonlyAdmin = &model.User{
		UserID: "22222222-2222-2222-2222-222222222222",
		Email:  "readonly@test.com",
		Role:   "HHS_READONLY_ADMIN",
	}
	issoUser = &model.User{
		UserID: "33333333-3333-3333-3333-333333333333",
		Email:  "isso@test.com",
		Role:   "ISSO",
	}
)

// --- SaveUser ---

func TestSaveUser_AdminAllowed(t *testing.T) {
	body := jsonBody(t, map[string]string{
		"email":    "new@test.com",
		"fullname": "New User",
		"role":     "ISSO",
	})
	r := httptest.NewRequest("POST", "/api/v1/users", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	SaveUser(w, r)
	// ADMIN should not get 403 (may get 500 due to no DB, that's fine)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestSaveUser_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{
		"email":    "new@test.com",
		"fullname": "New User",
		"role":     "ISSO",
	})
	r := httptest.NewRequest("POST", "/api/v1/users", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveUser(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveUser_ISSOForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{
		"email":    "new@test.com",
		"fullname": "New User",
		"role":     "ISSO",
	})
	r := httptest.NewRequest("POST", "/api/v1/users", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	SaveUser(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- DeleteUser ---

func TestRestoreUser_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/users/11111111-1111-1111-1111-111111111111/restore", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	RestoreUser(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRestoreUser_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/users/11111111-1111-1111-1111-111111111111/restore", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	RestoreUser(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteUser_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("DELETE", "/api/v1/users/11111111-1111-1111-1111-111111111111", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	DeleteUser(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- ListUsers ---

func TestListUsers_AdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	ListUsers(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestListUsers_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	ListUsers(w, r)
	// HHS_READONLY_ADMIN should get read access (not 403)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestListUsers_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	ListUsers(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- GetUserByID ---

func TestGetUserByID_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users/11111111-1111-1111-1111-111111111111", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	GetUserByID(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestGetUserByID_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users/11111111-1111-1111-1111-111111111111", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	GetUserByID(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveScore ---

func TestSaveScore_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"fismasystemid":    1,
		"functionoptionid": 1,
		"notes":            "test",
		"datacallid":       1,
	})
	r := httptest.NewRequest("POST", "/api/v1/scores", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveScore_ReadonlyAdminForbiddenEvenIfAssigned(t *testing.T) {
	// An HHS_READONLY_ADMIN assigned to a FISMA system should still be forbidden from saving
	assignedReadonly := &model.User{
		UserID:               "22222222-2222-2222-2222-222222222222",
		Email:                "readonly@test.com",
		Role:                 "HHS_READONLY_ADMIN",
		AssignedFismaSystems: []*int32{int32Ptr(1)},
	}
	body := jsonBody(t, map[string]any{
		"fismasystemid":    1,
		"functionoptionid": 1,
		"notes":            "test",
		"datacallid":       1,
	})
	r := httptest.NewRequest("POST", "/api/v1/scores", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, assignedReadonly)
	w := httptest.NewRecorder()

	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveScore_ISSONotAssignedForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"fismasystemid":    999,
		"functionoptionid": 1,
		"notes":            "test",
		"datacallid":       1,
	})
	r := httptest.NewRequest("POST", "/api/v1/scores", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveFismaSystemTargetMaturity (#398) ---

func TestSaveFismaSystemTargetMaturity_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"target_maturity_tier":          "Advanced",
		"target_maturity_justification": "should never write",
	})
	r := httptest.NewRequest("PUT", "/api/v1/fismasystems/1/target-maturity", body)
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{"fismasystemid": "1"})
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveFismaSystemTargetMaturity(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveFismaSystemTargetMaturity_ISSONotAssignedForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"target_maturity_tier":          "Advanced",
		"target_maturity_justification": "should never write",
	})
	r := httptest.NewRequest("PUT", "/api/v1/fismasystems/999/target-maturity", body)
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{"fismasystemid": "999"})
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	SaveFismaSystemTargetMaturity(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestSystemDelegate_ForbiddenNonAnswerSurfaces pins the delegate invariant
// (#455): a delegate may reach nothing an ISSO can that is not a data-call
// answer. The carve-out is enforced by explicit IsSystemDelegate() rejections
// rather than a central guard, so this table is the loud tripwire - if a future
// PR adds an ISSO/ISSM-writable non-answer surface without guarding it, add a row
// here and it will fail until the guard is in place.
//
// The delegate is assigned to the target system on purpose: assignment must NOT
// grant these surfaces, so a passing row proves the carve-out fires on the role,
// not merely on a missing assignment.
func TestSystemDelegate_ForbiddenNonAnswerSurfaces(t *testing.T) {
	// Identity matches the SYSTEM_DELEGATE row in _test_data_empire.sql so the
	// delegate is the same conceptual user across this unit test and the Emberfall
	// E2E. (The value is inert here - this test never hits the DB - but a shared
	// identity avoids the confusion of a mismatched or seed-colliding UUID.)
	delegate := &model.User{
		UserID:               "55555555-5555-4555-8555-555555555555",
		Email:                "Delegate.User@nowhere.xyz",
		Role:                 "SYSTEM_DELEGATE",
		AssignedFismaSystems: []*int32{int32Ptr(1)},
	}

	cases := []struct {
		name    string
		handler http.HandlerFunc
		request func() *http.Request
	}{
		{
			name:    "target maturity write",
			handler: SaveFismaSystemTargetMaturity,
			request: func() *http.Request {
				body := jsonBody(t, map[string]any{
					"target_maturity_tier":          "Advanced",
					"target_maturity_justification": "should never write",
				})
				r := httptest.NewRequest("PUT", "/api/v1/fismasystems/1/target-maturity", body)
				r.Header.Set("Content-Type", "application/json")
				return mux.SetURLVars(r, map[string]string{"fismasystemid": "1"})
			},
		},
		// Add a row when a new ISSO/ISSM-writable non-answer surface lands, and
		// add the matching IsSystemDelegate() guard to its handler.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := withUser(tc.request(), delegate)
			w := httptest.NewRecorder()
			tc.handler(w, r)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"delegate must be forbidden from non-answer surface %q", tc.name)
		})
	}
}

// --- ListScores ---

func TestListScores_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/scores", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	ListScores(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// --- GetScoresAggregate ---

func TestGetScoresAggregate_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/scores/aggregate", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	GetScoresAggregate(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// --- SaveDataCallFismaSystem ---

func TestSaveDataCallFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/datacalls/1/fismasystems/1", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveDataCallFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveDataCallFismaSystem_ReadonlyAdminForbiddenEvenIfAssigned(t *testing.T) {
	assignedReadonly := &model.User{
		UserID:               "22222222-2222-2222-2222-222222222222",
		Email:                "readonly@test.com",
		Role:                 "HHS_READONLY_ADMIN",
		AssignedFismaSystems: []*int32{int32Ptr(1)},
	}
	r := httptest.NewRequest("PUT", "/api/v1/datacalls/1/fismasystems/1", nil)
	r = withUser(r, assignedReadonly)
	w := httptest.NewRecorder()

	SaveDataCallFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveFismaSystem ---

func TestSaveFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"fismauid":     "12345678-1234-4abc-8def-123456789abc",
		"fismaacronym": "TEST",
		"fismaname":    "Test System",
	})
	r := httptest.NewRequest("POST", "/api/v1/fismasystems", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- DeleteFismaSystem ---

func TestDeleteFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("DELETE", "/api/v1/fismasystems/1", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	DeleteFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReactivateFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/fismasystems/1/reactivate", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	ReactivateFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReactivateFismaSystem_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/fismasystems/1/reactivate", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	ReactivateFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- ListFismaSystems ---

func TestListFismaSystems_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/fismasystems", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	ListFismaSystems(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestListFismaSystems_ISSOScopedToAssigned(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/fismasystems", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	ListFismaSystems(w, r)
	// ISSO should not get forbidden, just scoped results
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// --- SaveDataCall ---

func TestSaveDataCall_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"datacall": "FY2025 Q1",
		"deadline": "2025-03-31T17:59:59Z",
	})
	r := httptest.NewRequest("POST", "/api/v1/datacalls", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveDataCall(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveQuestion ---

func TestSaveQuestion_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"question":    "Test question?",
		"notesprompt": "Provide details",
		"pillarid":    1,
		"order":       1,
	})
	r := httptest.NewRequest("POST", "/api/v1/questions", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveQuestion(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveFunction ---

func TestSaveFunction_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"function":    "Test Function",
		"description": "Test description",
	})
	r := httptest.NewRequest("POST", "/api/v1/functions", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveFunction(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- ListUserFismaSystems ---

func TestListUserFismaSystems_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users/11111111-1111-1111-1111-111111111111/assignedfismasystems", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	ListUserFismaSystems(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestListUserFismaSystems_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/users/11111111-1111-1111-1111-111111111111/assignedfismasystems", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	ListUserFismaSystems(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- CreateUserFismaSystem ---

func TestCreateUserFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"fismasystemid": 1,
	})
	r := httptest.NewRequest("POST", "/api/v1/users/11111111-1111-1111-1111-111111111111/assignedfismasystems", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	CreateUserFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- DeleteUserFismaSystem ---

func TestDeleteUserFismaSystem_ReadonlyAdminForbidden(t *testing.T) {
	r := httptest.NewRequest("DELETE", "/api/v1/users/11111111-1111-1111-1111-111111111111/assignedfismasystems/1", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	DeleteUserFismaSystem(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- GetEvents ---

func TestGetEvents_AdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/events", nil)
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	GetEvents(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestGetEvents_ReadonlyAdminAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/events", nil)
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	GetEvents(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestGetEvents_ISSOForbidden(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/events", nil)
	r = withUser(r, issoUser)
	w := httptest.NewRecorder()

	GetEvents(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveMassEmail ---

func TestSaveMassEmail_ReadonlyAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{
		"subject": "Test",
		"body":    "Test body",
		"group":   "ALL",
	})
	r := httptest.NewRequest("POST", "/api/v1/massemails", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, readonlyAdmin)
	w := httptest.NewRecorder()

	SaveMassEmail(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// helper
func int32Ptr(i int32) *int32 {
	return &i
}

func TestDeleteUser_SelfDeleteRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+adminUser.UserID, nil)
	r = mux.SetURLVars(r, map[string]string{"userid": adminUser.UserID})
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	DeleteUser(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, auth.CodeSelfDeleteForbidden, body.Code)
	assert.NotEmpty(t, body.Error)
}

func TestDeleteUser_OtherDeleteSucceeds(t *testing.T) {
	otherIDLower := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	target := &model.User{
		UserID: otherIDLower,
		Email:  "target@empire.test",
		Role:   "ISSO",
	}

	prevFind, prevDel := findUserByID, deleteUser
	findUserByID = func(_ context.Context, _ string) (*model.User, error) {
		return target, nil
	}
	deleteUser = func(_ context.Context, _ string) error {
		return nil
	}
	t.Cleanup(func() {
		findUserByID = prevFind
		deleteUser = prevDel
	})

	
	variants := map[string]string{
		"lower-cased": otherIDLower,
		"upper-cased": strings.ToUpper(otherIDLower),
	}
	for name, pathID := range variants {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+pathID, nil)
			r = mux.SetURLVars(r, map[string]string{"userid": pathID})
			r = withUser(r, adminUser)
			w := httptest.NewRecorder()

			DeleteUser(w, r)

			assert.Equal(t, http.StatusNoContent, w.Code)
		})
	}
}

func TestDeleteUser_SelfDeleteRejected_CaseInsensitive(t *testing.T) {
	// Use a UUID with hex letters so case-folding is observable. Local
	// fixture rather than the shared adminUser (whose UserID is all digits).
	callerID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	caller := &model.User{
		UserID: callerID,
		Email:  "case@empire.test",
		Role:   "OWNER",
	}
	variants := map[string]string{
		"upper-cased": strings.ToUpper(callerID),
		"mixed-case":  "F47ac10B-58CC-4372-A567-0e02B2c3D479",
	}
	for name, pathID := range variants {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+pathID, nil)
			r = mux.SetURLVars(r, map[string]string{"userid": pathID})
			r = withUser(r, caller)
			w := httptest.NewRecorder()

			DeleteUser(w, r)

			assert.Equal(t, http.StatusForbidden, w.Code)
			var body struct {
				Code string `json:"code"`
			}
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, auth.CodeSelfDeleteForbidden, body.Code)
		})
	}
}

func TestDeleteUser_MissingIDIsNotFound(t *testing.T) {
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/users", nil)
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	DeleteUser(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- GetScoresProgress ---

// int32PtrAuthz keeps the OpDiv fixture below readable without pulling in a
// shared helper from another test file.
func int32PtrAuthz(v int32) *int32 { return &v }

// TestScopeScoreProgressInput pins the role matrix the progress endpoint
// applies to its query input. This is the security boundary for /scores/
// progress: the SQL builder is tested separately, so what matters here is
// that each tier populates (or leaves empty) exactly the scope fields the
// builder keys on.
func TestScopeScoreProgressInput(t *testing.T) {
	t.Run("OwnerUnrestricted", func(t *testing.T) {
		input := model.FindScoreProgressInput{}
		scopeScoreProgressInput(adminUser, &input)

		assert.False(t, input.RestrictToOpDivIDs, "OWNER must not be OpDiv-restricted")
		assert.Empty(t, input.OpDivIDs)
		assert.Nil(t, input.UserID, "OWNER must not be limited to assigned systems")
	})

	t.Run("HHSReadonlyAdminUnrestricted", func(t *testing.T) {
		input := model.FindScoreProgressInput{}
		scopeScoreProgressInput(readonlyAdmin, &input)

		assert.False(t, input.RestrictToOpDivIDs, "HHS_READONLY_ADMIN has unscoped read")
		assert.Empty(t, input.OpDivIDs)
		assert.Nil(t, input.UserID)
	})

	t.Run("OpDivAdminScopedToGrants", func(t *testing.T) {
		opdivAdmin := &model.User{
			UserID:           "44444444-4444-4444-4444-444444444444",
			Role:             "OPDIV_ADMIN",
			AssignedOpDivIDs: []*int32{int32PtrAuthz(7), int32PtrAuthz(9)},
		}
		input := model.FindScoreProgressInput{}
		scopeScoreProgressInput(opdivAdmin, &input)

		assert.True(t, input.RestrictToOpDivIDs, "OPDIV_ADMIN must be OpDiv-restricted")
		assert.Equal(t, []int32{7, 9}, input.OpDivIDs)
		assert.Nil(t, input.UserID, "OpDiv tier scopes by OpDiv, not by assignment")
	})

	t.Run("OpDivAdminWithNoGrantsFailsClosed", func(t *testing.T) {
		opdivAdmin := &model.User{
			UserID: "55555555-5555-5555-5555-555555555555",
			Role:   "OPDIV_READONLY_ADMIN",
		}
		input := model.FindScoreProgressInput{}
		scopeScoreProgressInput(opdivAdmin, &input)

		assert.True(t, input.RestrictToOpDivIDs)
		assert.Empty(t, input.OpDivIDs, "no grants must leave the id list empty so the query fails closed")
	})

	t.Run("ISSOScopedToAssignedSystems", func(t *testing.T) {
		input := model.FindScoreProgressInput{}
		scopeScoreProgressInput(issoUser, &input)

		assert.False(t, input.RestrictToOpDivIDs)
		if assert.NotNil(t, input.UserID, "ISSO must be limited to their assigned systems") {
			assert.Equal(t, issoUser.UserID, *input.UserID)
		}
	})
}

// TestGetScoresProgress_MissingDataCall_BadRequest exercises the handler end
// to end without a database: FindScoreProgress validates before touching the
// pool, so a request missing the required datacallid must come back 400 for
// every tier that can reach the endpoint.
func TestGetScoresProgress_MissingDataCall_BadRequest(t *testing.T) {
	for name, user := range map[string]*model.User{
		"Owner":         adminUser,
		"ReadonlyAdmin": readonlyAdmin,
		"ISSO":          issoUser,
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/scores/progress", nil)
			r = withUser(r, user)
			w := httptest.NewRecorder()

			GetScoresProgress(w, r)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
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
		Role:   "ADMIN",
	}
	readonlyAdmin = &model.User{
		UserID: "22222222-2222-2222-2222-222222222222",
		Email:  "readonly@test.com",
		Role:   "READONLY_ADMIN",
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
	// READONLY_ADMIN should get read access (not 403)
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
	// A READONLY_ADMIN assigned to a FISMA system should still be forbidden from saving
	assignedReadonly := &model.User{
		UserID:               "22222222-2222-2222-2222-222222222222",
		Email:                "readonly@test.com",
		Role:                 "READONLY_ADMIN",
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
		Role:                 "READONLY_ADMIN",
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

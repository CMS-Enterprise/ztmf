package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const grantUserID = "11111111-1111-4111-8111-111111111111"

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

package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The model-layer test in internal/model pins that FindFismaSystem CAN resolve
// the ISSO display name. This one pins that GetFismaSystem actually ASKS it to.
// Without this, dropping ResolveISSOName from the handler leaves the model test
// green while ztmf#510 silently regresses: the single-system GET would go back
// to returning the raw column and disagreeing with the list.
//
// Seed system 1002 carries isso_name NULL with an issoemail matching the
// Admiral Piett user record, so a resolved response can only come from the
// COALESCE fallback.
func TestGetFismaSystemResolvesISSONameIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	r := httptest.NewRequest("GET", "/api/v1/fismasystems/1002", nil)
	r = mux.SetURLVars(r, map[string]string{"fismasystemid": "1002"})
	r = withUser(r, adminUser)
	w := httptest.NewRecorder()

	GetFismaSystem(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			FismaSystemID int32   `json:"fismasystemid"`
			ISSOName      *string `json:"isso_name"`
			ISSOEmail     *string `json:"issoemail"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int32(1002), resp.Data.FismaSystemID)

	require.NotNil(t, resp.Data.ISSOName,
		"the single-system GET must resolve the ISSO name from the user record "+
			"the way the list does (ztmf#510); a nil here means the handler "+
			"stopped setting ResolveISSOName")
	assert.Equal(t, "Admiral Piett", *resp.Data.ISSOName)
}

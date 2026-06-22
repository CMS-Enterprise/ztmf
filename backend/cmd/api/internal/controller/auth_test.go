package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// LookupIdP's found/not-found branches hit the database and are exercised by the
// Emberfall E2E suite. Here we cover the branch that never touches the DB: a
// missing email is a malformed request and must be rejected before any lookup.
func TestLookupIdP_MissingEmail(t *testing.T) {
	for _, q := range []string{"", "?email=", "?email=%20%20"} {
		r := httptest.NewRequest("GET", "/api/v1/auth/lookup"+q, nil)
		w := httptest.NewRecorder()

		LookupIdP(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code, "query %q", q)

		var resp map[string]any
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// No data payload on a rejected request, and no IdP leaked.
		assert.Nil(t, resp["data"])
	}
}

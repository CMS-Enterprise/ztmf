package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// A malformed query param decoded via gorilla/schema must surface as a 400
// (client error), not a 500. Regression for #420: the decode error fell through
// sanitizeErr to the default 500 branch, inflating 5xx metrics for what is the
// caller's mistake. Exercises the real decoder so the concrete error type
// (schema.MultiError wrapping a ConversionError) matches what handlers pass in.
func TestSanitizeErrMapsSchemaDecodeErrorTo400(t *testing.T) {
	var input struct {
		FismaSystemID int `schema:"fismasystemid"`
	}
	err := decoder.Decode(&input, url.Values{"fismasystemid": {"abc"}})
	assert.Error(t, err, "decoding a non-numeric int param should error")

	status, code, out := sanitizeErr(err)
	assert.Equal(t, 400, status)
	assert.Equal(t, "", code)
	assert.Equal(t, ErrInvalidQueryParam, out, "message must be the sanitized sentinel, not the raw schema error")
}

// An unknown query param (gorilla/schema rejects unknown keys by default) is
// likewise a client error -> 400.
func TestSanitizeErrMapsUnknownQueryKeyTo400(t *testing.T) {
	var input struct {
		FismaSystemID int `schema:"fismasystemid"`
	}
	err := decoder.Decode(&input, url.Values{"bogus": {"1"}})
	assert.Error(t, err)

	status, _, out := sanitizeErr(err)
	assert.Equal(t, 400, status)
	assert.Equal(t, ErrInvalidQueryParam, out)
}

// The rest of the mapping is unchanged; pin the key cases so the added schema
// branch can't accidentally shadow them.
func TestSanitizeErrMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid input -> 400", &model.InvalidInputError{}, 400},
		{"no data -> 404", model.ErrNoData, 404},
		{"not found -> 404", ErrNotFound, 404},
		{"forbidden -> 403", ErrForbidden, 403},
		{"past deadline -> 403", model.ErrPastDeadline, 403},
		{"not unique -> 400", model.ErrNotUnique, 400},
		{"db connection -> 503", model.ErrDbConnection, 503},
		{"unknown -> 500", errors.New("boom"), 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, _ := sanitizeErr(tc.err)
			assert.Equal(t, tc.wantStatus, status)
		})
	}
}

// The administrator-required rejection carries a machine-readable code the FE
// branches on (#467), like the auth middleware's ACCOUNT_NOT_PROVISIONED.
func TestSanitizeErrDelegateRequiresAdminCarriesCode(t *testing.T) {
	status, code, out := sanitizeErr(model.ErrDelegateRequiresAdmin)
	assert.Equal(t, 400, status)
	assert.Equal(t, auth.CodeDelegateRequiresAdmin, code)
	assert.Equal(t, model.ErrDelegateRequiresAdmin, out, "human-readable message is preserved alongside the code")
}

// The capability-off rejection is a 403 that also carries a code, so the FE can
// render an in-dialog guard rather than let the global auth interceptor swallow
// the bare 403 into a generic toast.
func TestSanitizeErrDelegatesNotEnabledCarriesCode(t *testing.T) {
	status, code, out := sanitizeErr(model.ErrDelegatesNotEnabled)
	assert.Equal(t, 403, status)
	assert.Equal(t, auth.CodeDelegateNotEnabled, code)
	assert.Equal(t, model.ErrDelegatesNotEnabled, out, "human-readable message is preserved alongside the code")
}

// pathInt32 replaced fmt.Sscan for path variables. Sscan's default verb honours
// an octal prefix, and every mux pattern here is [0-9]+ - so "0012" routed
// fine and then scanned as 10, silently addressing the wrong row on a PUT.
//
// The second return is what keeps an unusable id from reading as an absent one.
// "0" and anything past int32 are routable through [0-9]+ and must report
// ok=false, or a PUT falls through to Save's create branch.
func TestPathInt32(t *testing.T) {
	cases := []struct {
		in   string
		want int32
		ok   bool
	}{
		{"1", 1, true},
		{"7001", 7001, true},
		{"0012", 12, true}, // octal prefix under fmt.Sscan: was 10
		{"010", 10, true},  // was 8
		{"2147483647", 2147483647, true},
		{"0", 0, false},          // routable through [0-9]+, and no row has id 0
		{"0000", 0, false},       // same, zero-padded
		{"2147483648", 0, false}, // past int32
		{"99999999999", 0, false},
		{"99999999999999999999", 0, false},
		{"", 0, false},
		{"abc", 0, false},
		{"-5", -5, true}, // unreachable through [0-9]+; never a silent other-row id
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := pathInt32(c.in)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.ok, ok)
		})
	}
}

// An id that routes but cannot address a row must be a 404, not a create. The
// mux patterns are [0-9]+, so both "0" and an over-int32 value reach these
// handlers; before the two-value pathInt32 they parsed to 0, were taken for
// "no id in the path", and Save inserted a new row under the PUT's 204.
//
// No DB: the 404 is returned before any query, which is what makes this a unit
// test rather than an integration one.
func TestSaveHandlers_UnusablePathIDIsNotACreate(t *testing.T) {
	owner := &model.User{Role: "OWNER"}

	handlers := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		varName string
		path    string
		body    map[string]any
	}{
		{
			name: "SaveQuestion", handler: SaveQuestion, varName: "questionid", path: "/api/v1/questions/",
			body: map[string]any{"question": "q?", "notesprompt": "prompt", "pillarid": 1},
		},
		{
			name: "SaveFunction", handler: SaveFunction, varName: "functionid", path: "/api/v1/functions/",
			body: map[string]any{"function": "fn", "description": "d", "datacenterenvironment": "AWS", "questionid": 1},
		},
		{
			name: "SaveFunctionOption", handler: SaveFunctionOption, varName: "functionoptionid", path: "/api/v1/functionoptions/",
			body: map[string]any{"score": 1, "optionname": "Traditional", "description": "d"},
		},
	}

	for _, h := range handlers {
		for _, id := range []string{"0", "99999999999"} {
			t.Run(h.name+" PUT "+id, func(t *testing.T) {
				r := httptest.NewRequest("PUT", h.path+id, jsonBody(t, h.body))
				r = mux.SetURLVars(r, map[string]string{h.varName: id})
				r = withUser(r, owner)
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				h.handler(w, r)
				assert.Equal(t, http.StatusNotFound, w.Code,
					"an unusable path id must 404, never fall through to a create")
			})
		}
	}
}

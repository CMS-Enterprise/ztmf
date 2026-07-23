package controller

import (
	"errors"
	"net/url"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
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

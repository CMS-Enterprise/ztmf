package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/schema"
)

// from gorilla/schema used to convert querystring params into structs.
// placed here because it caches struct meta-data
var decoder = schema.NewDecoder()

type response struct {
	Data any    `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
	// Set only for error responses that opt into
	// a typed code via sanitizeErr; omitted for both successful responses
	// and generic errors so existing consumers see the same shape.
	Code string `json:"code,omitempty"`
}

// respondOK writes an HTTP 200 with the JSON-wrapped data payload. Use for
// PUT-as-action endpoints (restore, reactivate) that return the updated entity
// rather than 204; respond() reserves PUT for in-place updates that drop the
// body.
func respondOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{Data: data})
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	w.Header().Set("Content-Type", "application/json")

	var status int

	switch r.Method {
	case "GET":
		status = 200
	case "POST":
		status = 201
	case "PUT":
		status = 204
	case "DELETE":
		// Return 200 with data if provided, otherwise 204
		if data != nil {
			status = 200
		} else {
			status = 204
		}
	default:
		// PATCH and any other verb: default to 200 so a handler that reaches
		// respond() on a success path never emits WriteHeader(0). (PATCH success
		// paths that want the body use respondOK; this is the defensive floor.)
		status = 200
	}

	res := response{
		Data: data,
	}

	if err == nil && (data == nil && status != 204) {
		err = ErrNotFound
	}

	if err != nil {
		var code string
		status, code, err = sanitizeErr(err)
		switch e := err.(type) {
		case *model.InvalidInputError:
			res.Data = e.Data()
		}
		res.Err = err.Error()
		res.Code = code
	}

	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(res)
}

func getJSON(r io.Reader, dest any) error {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	return d.Decode(dest)
}

func parseRFC3339(dateStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, dateStr)
}

// sanitizeErr maps an error to an HTTP status code and optional typed code string.
// error is last so the signature follows Go convention (status, code, err).
func sanitizeErr(err error) (int, string, error) {
	switch err.(type) {
	case *model.InvalidInputError:
		return 400, "", err
	case schema.MultiError, schema.ConversionError, *schema.ConversionError,
		schema.UnknownKeyError, *schema.UnknownKeyError,
		schema.EmptyFieldError, *schema.EmptyFieldError:
		// A gorilla/schema decode failure means the caller sent a query param
		// that can't be parsed into the input struct (e.g. fismasystemid=abc).
		// That's a client error across every GET list handler using the shared
		// decoder.Decode + respond() pattern, so surface a clean 400 rather than
		// falling through to a 500 that inflates 5xx metrics. (#420)
		return 400, "", ErrInvalidQueryParam
	}

	var (
		status int
		code   string
	)

	switch {
	case errors.Is(err, model.ErrNoData):
		err = ErrNotFound
		fallthrough
	case errors.Is(err, ErrNotFound):
		status = 404
	case errors.Is(err, ErrSelfDelete):
		status = 403
		code = auth.CodeSelfDeleteForbidden
	case errors.Is(err, ErrForbidden),
		errors.Is(err, model.ErrPastDeadline):
		status = 403
	case errors.Is(err, model.ErrDelegatesNotEnabled):
		// Capability-off for the OpDiv. 403 with a code so the FE can render an
		// in-dialog guard instead of the global interceptor swallowing the bare
		// 403 into a generic toast.
		status = 403
		code = auth.CodeDelegateNotEnabled
	case errors.Is(err, model.ErrDelegateRequiresAdmin):
		// Well-formed request, but the target account needs an administrator. 400
		// (not 403) so it does not collide with the FE's global auth handling; the
		// FE branches on the code, not the status.
		status = 400
		code = auth.CodeDelegateRequiresAdmin
	case errors.Is(err, model.ErrNotUnique),
		errors.Is(err, ErrMalformed),
		errors.Is(err, model.ErrNotesTooLong),
		errors.Is(err, model.ErrNoReference):
		status = 400
	case errors.Is(err, model.ErrDbConnection):
		err = ErrServiceUnavailable
		status = 503
	case errors.Is(err, model.ErrTooMuchData):
		fallthrough
	default:
		status = 500
		log.Printf("unknown error %s\n", err)
		err = ErrServer
	}

	return status, code, err
}

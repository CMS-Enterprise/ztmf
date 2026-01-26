package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/schema"
)

// from gorilla/schema used to convert querystring params into structs.
// placed here because it caches struct meta-data
var decoder = schema.NewDecoder()

type response struct {
	Data any    `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	w.Header().Set("Content-Type", "application/json")

	var status int

	switch r.Method {
	case "GET":
		status = 200
	case "POST":
		status = 201
	case "PUT", "DELETE":
		status = 204
	}

	res := response{
		Data: data,
	}

	if err == nil && (data == nil && status != 204) {
		err = ErrNotFound
	}

	if err != nil {
		status, err = sanitizeErr(err)
		switch e := err.(type) {
		case *model.InvalidInputError:
			res.Data = e.Data()
		}
		res.Err = err.Error()
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

func sanitizeErr(err error) (int, error) {
	switch err.(type) {
	case *model.InvalidInputError:
		return 400, err
	}

	var status int

	switch {
	case errors.Is(err, model.ErrNoData):
		err = ErrNotFound
		fallthrough
	case errors.Is(err, ErrNotFound):
		status = 404
	case errors.Is(err, ErrForbidden),
		errors.Is(err, model.ErrPastDeadline):
		status = 403
	case errors.Is(err, model.ErrNotUnique),
		errors.Is(err, ErrMalformed),

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

	return status, err
}

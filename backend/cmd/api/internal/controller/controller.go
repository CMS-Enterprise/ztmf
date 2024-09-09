package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

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

	if err == nil && data == nil {
		err = ErrNotFound
	}

	if err != nil {
		switch {
		case errors.Is(err, model.ErrNoData):
			status = 404
			err = ErrNotFound
		case errors.Is(err, ErrForbidden):
			status = 403
		case errors.Is(err, &model.InvalidInputError{}), errors.Is(err, model.ErrNotUnique):
			status = 400
		case errors.Is(err, model.ErrDbConnection):
			err = ErrServiceUnavailable
			status = 503
		case errors.Is(err, model.ErrTooMuchData):
			fallthrough
		default:
			status = 500
			err = ErrServer
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

	err := d.Decode(dest)
	if err != nil {
		return err
	}

	// // optional extra check
	// if d.More() {
	// 	http.Error(rw, "extraneous data after JSON object", http.StatusBadRequest)
	// 	return
	// }
	return nil
}

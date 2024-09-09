package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		err = &NotFoundError{}
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &NotFoundError{}
		}

		if errors.Is(err, pgx.ErrTooManyRows) || errors.Is(err, pgx.ErrTxClosed) || errors.Is(err, pgx.ErrTxCommitRollback) {
			err = &ServerError{}
		}

		switch err.(type) {
		case *pgconn.PgError:
			status = 500
			err = &ServerError{}
		case *ForbiddenError:
			status = 403
		case *InvalidInputError:
			status = 400
		case *NotFoundError:
			status = 404
		case *ServerError:
			status = 500
		default:
			status = 500
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

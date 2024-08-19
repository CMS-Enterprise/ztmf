package controller

import (
	"encoding/json"
	"io"
	"net/http"
)

// TODO: refactor to provide way of sending 201 vs 200
func respond(w http.ResponseWriter, data any, err error) {
	w.Header().Set("Content-Type", "application/json")
	status := 200

	if err == nil && data == nil {
		err = &NotFoundError{}
	}

	if err != nil {
		if err.Error() == "no rows in result set" {
			err = &NotFoundError{}
		}
		switch err.(type) {
		case *ForbiddenError:
			status = 403
		case *InvalidInputError:
			status = 400
		case *NotFoundError:
			status = 404
		default:
			status = 500
		}
		data = map[string]any{
			"error": err.Error(),
		}
	}
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(data)
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

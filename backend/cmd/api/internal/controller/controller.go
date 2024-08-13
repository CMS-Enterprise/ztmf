package controller

import (
	"encoding/json"
	"net/http"
)

func respond(w http.ResponseWriter, data any, err error) {
	w.Header().Set("Content-Type", "application/json")
	status := 200

	if err != nil {
		switch err.(type) {
		case *ForbiddenError:
			status = 403
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

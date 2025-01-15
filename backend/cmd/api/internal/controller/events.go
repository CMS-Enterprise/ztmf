package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

func GetEvents(w http.ResponseWriter, r *http.Request) {
	findEventsInput := &model.FindEventsInput{}

	v := r.URL.Query()
	err := decoder.Decode(findEventsInput, v)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	events, err := model.FindEvents(r.Context(), findEventsInput)

	respond(w, r, events, err)
}

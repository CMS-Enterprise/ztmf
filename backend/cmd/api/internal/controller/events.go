package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

func GetEvents(w http.ResponseWriter, r *http.Request) {
	findEventsInput := &model.FindEventsInput{}
	err := decoder.Decode(findEventsInput, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	events, err := model.FindEvents(r.Context(), findEventsInput)

	respond(w, r, events, err)
}

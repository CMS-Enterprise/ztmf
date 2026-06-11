package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

func GetEvents(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	// The audit trail spans every OpDiv and events carry no opdiv_id to scope
	// on, so it is restricted to unscoped admins (OWNER / HHS_ADMIN /
	// HHS_READONLY_ADMIN). OpDiv-scoped tiers get 403 rather than a cross-OpDiv
	// audit view.
	if !user.HasUnscopedRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	findEventsInput := &model.FindEventsInput{}
	err := decoder.Decode(findEventsInput, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	events, err := model.FindEvents(r.Context(), findEventsInput)

	respond(w, r, events, err)
}

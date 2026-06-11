package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// GetEvents godoc
//
//	@Summary	List audit-trail events
//	@Tags		events
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid			query		string	false	"Filter by initiating user ID"
//	@Param		fismasystemid	query		integer	false	"Filter by FISMA system ID (payload)"
//	@Param		scoreid			query		integer	false	"Filter by score ID (payload)"
//	@Param		datacallid		query		integer	false	"Filter by data call ID (payload)"
//	@Param		questionid		query		integer	false	"Filter by question ID (payload)"
//	@Success	200	{object}	apiResponse[[]model.Event]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/events [get]
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

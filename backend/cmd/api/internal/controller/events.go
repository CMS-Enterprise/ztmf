package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

//	@Summary	List audit-trail events
//	@Tags		events
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid					query		string	false	"Filter by initiating user ID"
//	@Param		action					query		string	false	"Filter by action: created, updated, or deleted"
//	@Param		resource				query		string	false	"Filter by affected resource (table name)"
//	@Param		payload.fismasystemid	query		integer	false	"Filter by FISMA system ID referenced in the event payload"
//	@Param		payload.scoreid			query		integer	false	"Filter by score ID referenced in the event payload"
//	@Param		payload.datacallid		query		integer	false	"Filter by data call ID referenced in the event payload"
//	@Param		payload.questionid		query		integer	false	"Filter by question ID referenced in the event payload"
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

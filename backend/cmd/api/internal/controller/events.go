package controller

import (
	"net/http"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

//	@Summary		Record a questionnaire question view
//	@Description	Appends a 'viewed' event marking that the caller opened a questionnaire question, so time-spent analytics can bound how long the question was worked on before the next view. Editor-vs-viewer is derived server-side (from the caller's role and the data call deadline), not sent by the client. Recorded for any caller who can see the system; a caller may only record views for a system they have a relationship to (admins any, OpDiv-scoped admins their OpDivs, ISSO/ISSM/SYSTEM_DELEGATE their assigned systems). Returns 404 if the system or data call does not exist.
//	@Tags		events
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		body	body		model.QuestionViewInput	true	"Question view to record"
//	@Success	204		"No Content"
//	@Failure	400		{object}	apiResponse[any]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	404		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/events/view [post]
func RecordQuestionView(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	input := model.QuestionViewInput{}
	if err := getJSON(r.Body, &input); err != nil {
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Reject a malformed body up front, before any access/data-call lookups.
	if err := input.Validate(); err != nil {
		respond(w, r, nil, err)
		return
	}

	// A caller may only record views for a system they could SEE (read scope),
	// so analytics never accrue for a system the user has no relationship to.
	// CanAccessFismaSystem needs the system's OpDiv for the OpDiv-scoped tiers,
	// so load the system first; unscoped-read and assigned-system callers
	// short-circuit before that.
	if err := guardViewFismaSystem(r.Context(), user, input.FismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	// Load the data call: this validates it exists (a bad/unknown id is rejected
	// rather than recorded) and provides the deadline used to classify the view.
	dc, err := model.FindDataCallByID(r.Context(), input.DataCallID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// Derive read-only server-side; never trust a client-sent value. It decides
	// whether this view's dwell counts as viewer or editor time, so a client
	// must not be able to choose it. Mirrors the questionnaire's rule: a
	// read-only admin is always viewing, and any non-admin is viewing (not
	// editing) once the data call's deadline has passed.
	input.ReadOnly = user.IsReadOnlyAdmin() || (!user.IsAdmin() && time.Now().After(dc.Deadline))

	// On error let respond() map it to a status; on success write 204 directly
	// (respond() would treat a nil-body POST as 201-with-empty-body, and this
	// fire-and-forget ping has no entity to return).
	if err := model.RecordQuestionView(r.Context(), input); err != nil {
		respond(w, r, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

//	@Summary	List audit-trail events
//	@Tags		events
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid					query		string	false	"Filter by initiating user ID"
//	@Param		action					query		string	false	"Filter by action: created, updated, deleted, or viewed"
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

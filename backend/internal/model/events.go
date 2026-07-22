package model

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/lann/builder"
)

type Event struct {
	UserID    string      `json:"userid"`    // who initiated the event
	Action    string      `json:"action"`    // the action they took
	Resource  string      `json:"type"`      // on what resource
	CreatedAt *time.Time  `json:"createdat"` // at what date and time
	Payload   interface{} `json:"payload"`   // incoming data
}

// json tags here are used when payload is marshaled into select Where argument (see FindEvents() )
type payload struct {
	UserID        *string `schema:"userid" json:"userid,omitempty"`
	FismaSystemID *int32  `schema:"fismasystemid" json:"fismasystemid,omitempty"`
	ScoreID       *int32  `schema:"scoreid" json:"scoreid,omitempty"`
	DataCallID    *int32  `schema:"datacallid" json:"datacallid,omitempty"`
	QuestionID    *int32  `schema:"questionid" json:"questionid,omitempty"`
	// ReadOnly records whether a 'viewed' event was made in a read-only session.
	// A pointer so it is omitted from non-view payloads and only stamped on
	// views: true attributes the dwell to viewer time, false to editor time.
	ReadOnly *bool `schema:"readonly" json:"readonly,omitempty"`
}

type FindEventsInput struct {
	UserID   *string  `schema:"userid" json:"userid,omitempty"`
	Action   *string  `schema:"action" json:"action,omitempty"`
	Resource *string  `schema:"resource" json:"resource,omitempty"`
	Payload  *payload `schema:"payload" json:"payload,omitempty"`
}

// recordEvent uses the provided SqlBuilder to determin what write operation was performed (create, update, delete), and
// records that along with current user ID, the resource being acted upon, and the payload for the event.
// The event payload is essentially the row that was inserted or updated, but in this case stored as JSONB.
//
// Error handling: the inner queryRow call logs but does not return its
// error to recordEvent's caller. The outer write that triggered this
// hook (e.g. scores INSERT) has already succeeded by the time we get
// here, so failing the response would lie about what is in the DB.
// Callers that need to confirm an event was actually written (for
// example, before stamping audit fields onto a response) must read
// back from the events table rather than trust this side-channel - see
// lookupScoreAudit in scores.go for the canonical pattern.
func recordEvent(ctx context.Context, sqlb SqlBuilder, res interface{}) {

	e := Event{
		Payload: res,
	}

	eventData := builder.GetMap(sqlb)

	switch sqlb.(type) {
	case squirrel.InsertBuilder:
		e.Action = "created"
		e.Resource = eventData["Into"].(string)
	case squirrel.UpdateBuilder:
		e.Action = "updated"
		e.Resource = eventData["Table"].(string)
	case squirrel.DeleteBuilder:
		e.Action = "deleted"
		e.Resource = eventData["From"].(string)
	default:
		return
	}

	if e.Resource == "events" {
		return
	}

	user := UserFromContext(ctx)
	if user == nil {
		return
	}

	// Fire-and-forget: the outer write already succeeded, so a failed event
	// insert must not fail the response (see the doc comment above). The error
	// is discarded here but logged inside queryRow.
	insertEvent(ctx, user.UserID, e.Action, e.Resource, e.Payload)
}

// insertEvent appends a single row to the events audit log. It is the shared
// write behind both recordEvent (the write-derived side-effect hook, which
// discards the error) and RecordQuestionView (an explicit, purpose-built event,
// which returns it). The insert flows through queryRow, whose recordEvent hook
// short-circuits on resource == "events", so recording an event never recurses
// into recording another.
func insertEvent(ctx context.Context, userID, action, resource string, payload any) error {
	sqlb := stmntBuilder.
		Insert("events").
		Columns("userid", "action", "resource", "payload").
		Values(userID, action, resource, payload).
		Suffix("Returning *")

	_, err := queryRow(ctx, sqlb, pgx.RowToStructByName[Event])
	return err
}

// QuestionViewInput carries the identifiers for a questionnaire "viewed" event:
// which question, on which system, in which data call. userid is never taken
// from the client - it comes from the auth context in RecordQuestionView.
type QuestionViewInput struct {
	FismaSystemID int32 `json:"fismasystemid"`
	DataCallID    int32 `json:"datacallid"`
	QuestionID    int32 `json:"questionid"`
	// ReadOnly is true when the caller opened the question in a read-only
	// session. It decides whether this view's dwell counts as viewer time
	// (true) or editor time (false) in the time-spent analytics.
	ReadOnly bool `json:"readonly"`
}

func (i QuestionViewInput) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if i.FismaSystemID == 0 {
		err.data["fismasystemid"] = "required"
	}
	if i.DataCallID == 0 {
		err.data["datacallid"] = "required"
	}
	if i.QuestionID == 0 {
		err.data["questionid"] = "required"
	}

	if len(err.data) > 0 {
		return &err
	}
	return nil
}

// RecordQuestionView appends a 'viewed' event to the audit log marking that the
// current user opened a questionnaire question. Time-spent analytics pair each
// view with the next event by the same user in the same system+data call (a
// save, or the next view) to bound how long the question was worked on.
//
// Unlike recordEvent - which fires as a side effect of a write and derives its
// action from the SqlBuilder shape - this records an explicit event: a view is
// not a table mutation, so no write path produces it. It shares recordEvent's
// insertEvent primitive but supplies its own action ('viewed') and resource
// ('questionnaire', not 'public.scores', so these rows never touch the
// score-audit lookups), and it returns the insert error so the caller can
// surface a failure rather than swallow it.
func RecordQuestionView(ctx context.Context, input QuestionViewInput) error {
	if err := input.validate(); err != nil {
		return err
	}

	user := UserFromContext(ctx)
	if user == nil {
		// Every route that reaches here is behind auth.Middleware, so a nil
		// user is not reachable in practice; mirror recordEvent and skip
		// rather than fabricate an event with no initiator.
		return nil
	}

	p := payload{
		FismaSystemID: &input.FismaSystemID,
		DataCallID:    &input.DataCallID,
		QuestionID:    &input.QuestionID,
		ReadOnly:      &input.ReadOnly,
	}

	return insertEvent(ctx, user.UserID, "viewed", "questionnaire", p)
}

func FindEvents(ctx context.Context, input *FindEventsInput) ([]*Event, error) {

	sqlb := stmntBuilder.
		Select("*").
		From("events")

	if input.UserID != nil {
		sqlb = sqlb.Where("userid=?", input.UserID)
	}

	if input.Resource != nil {
		sqlb = sqlb.Where("resource=?", input.Resource)
	}

	if input.Action != nil {
		sqlb = sqlb.Where("action=?", input.Action)
	}

	if input.Payload != nil {
		p, err := json.Marshal(input.Payload)
		if err != nil {
			return nil, err
		}
		sqlb = sqlb.Where("payload @> ?", string(p))
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[Event])
}

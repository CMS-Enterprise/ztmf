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

// Event actions. These are the complete set of values that may appear in
// events.action, collected here so the Go writers agree by construction
// instead of by five hand-repeated string literals staying in sync.
//
// Treat these as stored data, not as an internal enum: the values are
// load-bearing well beyond this package, and several of the readers cannot
// import Go at all.
//
//   - migration 0048's status backfill selects on
//     `e.action IN ('created', 'updated')` as SQL literals, and it has already
//     run in every environment - the rows it wrote are permanent.
//   - the seed data (_test_data_empire.sql) repeats that same predicate to
//     keep scores.status agreeing with the events it seeds.
//   - the analyst queries in docs/timespent_queries.sql and the progress
//     query in scoreprogress.go filter on these literals inline.
//
// Renaming one of these constants' VALUES is therefore a data migration
// (backfill the events table, update every SQL predicate above), not a
// refactor. Renaming the Go identifier is free.
const (
	// eventActionCreated / eventActionUpdated / eventActionDeleted are derived
	// from the SqlBuilder shape by recordEvent, so every write that flows
	// through queryRow is attributed automatically.
	eventActionCreated = "created"
	eventActionUpdated = "updated"
	eventActionDeleted = "deleted"

	// eventActionViewed is recorded explicitly by RecordQuestionView. A view is
	// not a table mutation, so no write path produces it.
	eventActionViewed = "viewed"

	// eventActionImported marks provenance for score data loaded outside the
	// application - bulk imports and seed SQL, not a human answering a
	// questionnaire. Nothing in this package writes it today; the value lives
	// here so there is one authoritative home for it, because the readers that
	// exclude it (0048's backfill, the seed status-sync, the last-updated
	// lateral in scoreprogress.go) all depend on the exact spelling. A future
	// bulk importer must write this action rather than reusing the in-app
	// create/update path - see Score.Save.
	//
	// Having no Go writer is the point, so silence the unused check rather than
	// leaving the value defined only in SQL and prose. If a Go importer ever
	// does write it, staticcheck flags this directive as matching nothing and
	// it should be deleted.
	//lint:ignore U1000 defined as the authoritative spelling for SQL readers; no Go writer by design
	eventActionImported = "imported"
)

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
		e.Action = eventActionCreated
		e.Resource = eventData["Into"].(string)
	case squirrel.UpdateBuilder:
		e.Action = eventActionUpdated
		e.Resource = eventData["Table"].(string)
	case squirrel.DeleteBuilder:
		e.Action = eventActionDeleted
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

// QuestionViewInput carries the client-supplied identifiers for a questionnaire
// "viewed" event: which question, on which system, in which data call. userid
// comes from the auth context, and the editor/viewer (readOnly) classification
// is derived server-side - neither is part of this request shape.
type QuestionViewInput struct {
	FismaSystemID int32 `json:"fismasystemid"`
	DataCallID    int32 `json:"datacallid"`
	QuestionID    int32 `json:"questionid"`
}

// Validate is exported so the controller can reject a malformed body before
// doing any access/data-call lookups; RecordQuestionView also calls it as a
// backstop.
func (i QuestionViewInput) Validate() error {
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
// current user opened a questionnaire question. readOnly (derived server-side by
// the caller) classifies the dwell as viewer (true) or editor (false) time.
// Time-spent analytics pair each view with the NEXT VIEW by the same user in the
// same system+data call to bound how long the question was worked on (saves are
// not boundaries).
//
// Unlike recordEvent - which fires as a side effect of a write and derives its
// action from the SqlBuilder shape - this records an explicit event: a view is
// not a table mutation, so no write path produces it. It shares recordEvent's
// insertEvent primitive but supplies its own action ('viewed') and resource
// ('questionnaire', not 'public.scores', so these rows never touch the
// score-audit lookups), and it returns the insert error so the caller can
// surface a failure rather than swallow it.
func RecordQuestionView(ctx context.Context, input QuestionViewInput, readOnly bool) error {
	if err := input.Validate(); err != nil {
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
		ReadOnly:      &readOnly,
	}

	return insertEvent(ctx, user.UserID, eventActionViewed, "questionnaire", p)
}

// RecordLogin appends a "session / created" event for a user who has just
// completed authentication and been issued a session.
//
// Every other event in the log is a side effect of a write, so activity was
// only ever visible for users who CHANGED something. That left a specific blind
// spot: a privileged account that signs in and only reads looked identical to
// one that had never signed in at all, which is exactly the pair you need to
// tell apart to find a stale account.
//
// The caller is the post-OIDC session handler, not a request behind
// auth.Middleware, so the user is passed explicitly rather than read from
// context - at this point in the flow the session that would carry it has only
// just been minted.
//
// The payload is deliberately just the userid: the identity provider is already
// on the user row, and the interesting facts here are who and when, both of
// which the event row itself carries.
func RecordLogin(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	return insertEvent(ctx, userID, "created", "session", payload{UserID: &userID})
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

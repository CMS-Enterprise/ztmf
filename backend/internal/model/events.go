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
	EventID   int64       `json:"eventid"`   // identity column; the paging tiebreaker (migration 0058)
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
	// here so there is one authoritative home for it.
	//
	// Note which consumers actually depend on the exact spelling: the readers
	// that exclude imported rows (0048's backfill, the seed status-sync, the
	// last-updated lateral in scoreprogress.go) do NOT name this value at all -
	// they allowlist 'created'/'updated', so they would exclude an import under
	// any spelling. The sites that would silently drift on a respelling are the
	// WRITERS and the assertions over them: the seed INSERT in
	// _test_data_empire.sql and scoreprogress_integration_test.go, which read
	// back `action='imported'` to prove imported history stays not_started.
	//
	// A future bulk importer must write this action rather than reusing the
	// in-app create/update path - see Score.Save.
	//
	// Having no Go writer is the point, so silence the unused check rather than
	// leaving the value defined only in SQL and prose. Staticcheck will NOT
	// tell you when this directive goes stale - U1000 ignores are exempt from
	// its unmatched-directive check - so delete it by hand if a Go writer ever
	// appears.
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
	// Limit and Offset are unsigned so the shared query decoder rejects
	// negatives as a conversion error (a 400) without any range checks here.
	// Absent or zero Limit means the default; values above the cap clamp.
	Limit  *uint32 `schema:"limit" json:"limit,omitempty"`
	Offset *uint32 `schema:"offset" json:"offset,omitempty"`
	// From/To bound createdat inclusively. The shared decoder has no
	// time.Time converter, so the controller parses them from RFC3339 and
	// strips the keys from what the decoder sees; schema:"-" keeps the
	// decoder from ever trying.
	From *time.Time `schema:"-" json:"from,omitempty"`
	To   *time.Time `schema:"-" json:"to,omitempty"`
}

// Paging bounds for FindEvents. GET /events returned the entire table before
// the admin events page (ztmf#564); the cap guarantees no single request can
// do that again.
const (
	defaultEventsLimit uint32 = 50
	maxEventsLimit     uint32 = 500
)

// limit returns the page size to apply: the default when absent or zero, the
// cap when the caller asks for more, the caller's value otherwise.
func (i *FindEventsInput) limit() uint32 {
	switch {
	case i.Limit == nil || *i.Limit == 0:
		return defaultEventsLimit
	case *i.Limit > maxEventsLimit:
		return maxEventsLimit
	default:
		return *i.Limit
	}
}

func (i *FindEventsInput) offset() uint32 {
	if i.Offset == nil {
		return 0
	}
	return *i.Offset
}

// EventsPage is one page of the audit trail plus what a client needs to render
// paging controls. Total counts every event matching the filters, not just this
// page; Limit and Offset echo the values actually applied after defaulting and
// clamping, so a client can trust them for page math without re-deriving the
// rules.
type EventsPage struct {
	Events []*Event `json:"events"`
	Total  int64    `json:"total"`
	Limit  uint32   `json:"limit"`
	Offset uint32   `json:"offset"`
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
	return insertEvent(ctx, userID, eventActionCreated, "session", payload{UserID: &userID})
}

func FindEvents(ctx context.Context, input *FindEventsInput) (*EventsPage, error) {

	if input.From != nil && input.To != nil && input.From.After(*input.To) {
		return nil, &InvalidInputError{data: map[string]any{"from": "must not be after to"}}
	}

	// Marshaled once, ahead of the closure, so a marshal failure surfaces
	// before any query is built.
	var payloadJSON *string
	if input.Payload != nil {
		p, err := json.Marshal(input.Payload)
		if err != nil {
			return nil, err
		}
		s := string(p)
		payloadJSON = &s
	}

	// The page query and the count query must agree on the filters or Total
	// lies to the client's pager; one closure keeps them from drifting.
	where := func(sqlb squirrel.SelectBuilder) squirrel.SelectBuilder {
		if input.UserID != nil {
			sqlb = sqlb.Where("userid=?", input.UserID)
		}
		if input.Resource != nil {
			sqlb = sqlb.Where("resource=?", input.Resource)
		}
		if input.Action != nil {
			sqlb = sqlb.Where("action=?", input.Action)
		}
		if input.From != nil {
			sqlb = sqlb.Where("createdat >= ?", input.From)
		}
		if input.To != nil {
			sqlb = sqlb.Where("createdat <= ?", input.To)
		}
		if payloadJSON != nil {
			sqlb = sqlb.Where("payload @> ?", *payloadJSON)
		}
		return sqlb
	}

	limit, offset := input.limit(), input.offset()

	// eventid breaks createdat ties (bulk writers stamp identical
	// timestamps), making the order total so rows cannot swap across page
	// boundaries between requests. See migration 0058.
	sqlb := where(stmntBuilder.Select("*").From("events")).
		OrderBy("createdat DESC", "eventid DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	events, err := query(ctx, sqlb, pgx.RowToAddrOfStructByName[Event])
	if err != nil {
		return nil, err
	}
	if events == nil {
		// CollectRows yields nil for zero rows; an empty page must serialize
		// as [] rather than null.
		events = []*Event{}
	}

	total, err := queryRow(ctx, where(stmntBuilder.Select("COUNT(*)").From("events")), pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}

	return &EventsPage{Events: events, Total: *total, Limit: limit, Offset: offset}, nil
}

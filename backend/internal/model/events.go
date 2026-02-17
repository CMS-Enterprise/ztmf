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
}

type FindEventsInput struct {
	UserID   *string  `schema:"userid" json:"userid,omitempty"`
	Action   *string  `schema:"action" json:"action,omitempty"`
	Resource *string  `schema:"resource" json:"resource,omitempty"`
	Payload  *payload `schema:"payload" json:"payload,omitempty"`
}

// recordEvent uses the provided SqlBuilder to determin what write operation was performed (create, update, delete), and
// records that along with current user ID, the resource being acted upon, and the payload for the event.
// The event payload is essentially the row that was inserted or updated, but in this case stored as JSONB
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

	e.UserID = user.UserID

	sqlb = stmntBuilder.
		Insert("events").
		Columns("userid", "action", "resource", "payload").
		Values(e.UserID, e.Action, e.Resource, e.Payload).
		Suffix("Returning *")

	queryRow(ctx, sqlb, pgx.RowToStructByName[Event])
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

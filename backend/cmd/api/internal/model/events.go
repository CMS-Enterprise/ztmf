package model

import (
	"context"
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
	Payload   interface{} `json:"payload"`   // data
}

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

	e.UserID = user.UserID

	sqlb = stmntBuilder.
		Insert("events").
		Columns("userid", "action", "resource", "payload").
		Values(e.UserID, e.Action, e.Resource, e.Payload).
		Suffix("Returning *")

	queryRow(ctx, sqlb, pgx.RowToStructByName[Event])
}

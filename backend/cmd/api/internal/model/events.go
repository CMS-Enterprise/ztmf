package model

import (
	"context"
	"log"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lann/builder"
)

type Event struct {
	UserID    string                 `json:"userid"`    // who initiated the event
	Action    string                 `json:"action"`    // the action they took
	Resource  string                 `json:"type"`      // on what resource
	CreatedAt time.Time              `json:"createdat"` // at what date and time
	Payload   map[string]interface{} `json:"payload"`   // data
}

func recordEvent(ctx context.Context, sqlb SqlBuilder) {

	e := Event{
		CreatedAt: time.Now(),
		Payload:   map[string]interface{}{},
	}

	eventData := builder.GetMap(sqlb)

	switch sqlb.(type) {
	case squirrel.InsertBuilder:
		e.Action = "created"
		e.Resource = eventData["Into"].(string)
		// builder map: map[Columns:[email, fullname, role] Into:users PlaceholderFormat:{} Suffixes:[{sql:RETURNING userid, email, fullname, role args:[]}] Values:[[richard.jones55555555555@cms.hhs.gov Richard Jones ISSM]]]
		columns := eventData["Columns"].([]string)
		values := eventData["Values"].([][]any)[0]
		log.Printf("values: %#v\n", values)
		log.Printf("columns: %#v\n", columns)
		for i, col := range columns {
			log.Println(col, values[i])
			e.Payload[col] = values[i]
		}
	case squirrel.UpdateBuilder:
		e.Action = "updated"
		e.Resource = eventData["Table"].(string)
		// builder map: map[PlaceholderFormat:{} SetClauses:[{column:email value:richard.jones3@cms.hhs.gov} {column:fullname value:Richard Jones} {column:role value:ISSM}] Suffixes:[{sql:RETURNING userid, email, fullname, role args:[]}] Table:users WhereParts:[0xc0004d8000]]
	case squirrel.DeleteBuilder:
		e.Action = "deleted"
		e.Resource = eventData["Into"].(string)
	default:
		return
	}

	user := UserFromContext(ctx)

	e.UserID = user.UserID

	log.Printf("event: %+v\nbuilder map: %+v\n", e, eventData)
}

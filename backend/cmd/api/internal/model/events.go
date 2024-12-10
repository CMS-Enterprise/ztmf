package model

import (
	"time"
)

type Event struct {
	UserID    string                 `json:"userid"`    // who initiated the event
	Action    string                 `json:"action"`    // the action they took
	Resource  string                 `json:"type"`      // on what resource
	CreatedAt time.Time              `json:"createdat"` // at what date and time
	Payload   map[string]interface{} `json:"payload"`   // data
}

// func RecordEvent(ctx context.Context, sqlb SqlBuilder) {
// 	var (
// 		action, resource string
// 	)
// 	user := UserFromContext(ctx)

// 	switch sqlb.(type) {
// 	case squirrel.InsertBuilder:
// 		action = "created"
// 	case squirrel.UpdateBuilder:
// 		action = "updated"
// 	}

// 	e := Event{
// 		UserID:    user.UserID,
// 		Action:    action,
// 		Resource:  resource,
// 		CreatedAt: time.Now(),
// 	}

// 	builder.GetMap(sqlb)
// }

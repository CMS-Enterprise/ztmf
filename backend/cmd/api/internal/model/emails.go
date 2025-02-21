package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// MassEmail table is meant to hold a single row that is updated when emails are sent
// previous email data will be stored in the event history
// this prevents the duplicate storage of many records
// and there is no real value in accessing or modifying individual records
type MassEmail struct {
	MassEmailID int        `json:"massemailid"`
	DateSent    *time.Time `json:"datesent"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
}

func (e *MassEmail) Save(ctx context.Context) (*MassEmail, error) {
	sqlb := stmntBuilder.
		Update("massemails").
		Set("datesent", time.Now()).
		Set("subject", e.Subject).
		Set("body", e.Body).
		Where("massemailid=?", 1).
		Suffix("RETURNING *")

	return queryRow(ctx, sqlb, pgx.RowToStructByName[MassEmail])
}

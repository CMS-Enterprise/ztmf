package model

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var dataCallColumns = []string{"datacallid", "datacall", "datecreated", "deadline", "emailsubject", "emailbody", "emailsent"}

type DataCall struct {
	DataCallID   int32     `json:"datacallid"`
	DataCall     string    `json:"datacall"`
	DateCreated  time.Time `json:"datecreated"`
	Deadline     time.Time `json:"deadline"`
	EmailSubject *string   `json:"emailsubject"`
	EmailBody    *string   `json:"emailbody"`
	EmailSent    *string   `json:"emailsent"`
}

func (d *DataCall) fields() []any {
	return []any{&d.DataCallID, &d.DataCall, &d.DateCreated, &d.Deadline, &d.EmailSubject, &d.EmailBody, &d.EmailSent}
}

func (d *DataCall) Save(ctx context.Context) error {

	var (
		sqlb sqlBuilder
		err  error
	)

	// if valid, err := d.isValid(); !valid {
	// 	return err
	// }

	if d.DataCallID == 0 {
		sqlb = stmntBuilder.
			Insert("datacalls").
			Columns("datacall", "deadline", "emailsubject", "emailbody").
			Values(d.DataCall, d.Deadline, d.EmailSubject, d.EmailBody).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", "))
	} else {
		sqlb = stmntBuilder.
			Update("datacalls").
			Set("datacall", d.DataCall).
			Set("deadline", d.Deadline).
			Set("emailsubject", d.EmailSubject).
			Set("emailbody", d.EmailBody).
			Where("datacallid=?", d.DataCallID).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", "))
	}

	row, err := queryRow(ctx, sqlb)
	if err != nil {
		return trapError(err)
	}

	err = row.Scan(d.fields()...)

	return trapError(err)
}

func FindDataCalls(ctx context.Context) ([]*DataCall, error) {
	sqlb := stmntBuilder.Select(dataCallColumns...).
		From("datacalls").
		OrderBy("datecreated DESC")

	rows, err := query(ctx, sqlb)

	if err != nil {
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*DataCall, error) {
		datacall := DataCall{}
		err := row.Scan(datacall.fields()...)
		return &datacall, trapError(err)
	})
}

func FindDataCallByID(ctx context.Context, dataCallID int32) (*DataCall, error) {
	sqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid=?", dataCallID)

	row, err := queryRow(ctx, sqlb)
	if err != nil {
		return nil, trapError(err)
	}

	datacall := DataCall{}
	err = row.Scan(datacall.fields()...)

	return &datacall, err
}

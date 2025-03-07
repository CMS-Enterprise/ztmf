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

func (d *DataCall) Save(ctx context.Context) (*DataCall, error) {

	var sqlb SqlBuilder

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

	dataCall, err := queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
	if err != nil {
		return nil, err
	}

	go copyPreviousScores(dataCall.DataCallID)

	return dataCall, nil
}

func FindDataCalls(ctx context.Context) ([]*DataCall, error) {
	sqlb := stmntBuilder.Select(dataCallColumns...).
		From("datacalls").
		OrderBy("datecreated DESC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCall])
}

func FindDataCallByID(ctx context.Context, dataCallID int32) (*DataCall, error) {
	sqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid=?", dataCallID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
}

func findPreviousDataCall(dataCallID int32) (*DataCall, error) {
	// find the *previous* datacall
	// using dataCallID-1 would not suffice because it could have been deleted
	prevDcSqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid!=?", dataCallID).
		OrderBy("datacallid DESC"). // descending because the primary key is serial auto-incrementing
		Limit(1)

	return queryRow(context.TODO(), prevDcSqlb, pgx.RowToStructByName[DataCall])
}

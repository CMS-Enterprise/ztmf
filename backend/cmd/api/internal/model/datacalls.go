package model

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var dataCallColumns = []string{"datacallid", "datacall", "datecreated", "deadline"}

type DataCall struct {
	DataCallID  int32     `json:"datacallid"`
	DataCall    string    `json:"datacall"`
	DateCreated time.Time `json:"datecreated"`
	Deadline    time.Time `json:"deadline"`
}

func (d *DataCall) fields() []any {
	return []any{&d.DataCallID, &d.DataCall, &d.DateCreated, &d.Deadline}
}

func (d *DataCall) Save(ctx context.Context) error {

	var (
		sql       string
		boundArgs []any
		err       error
	)

	// if valid, err := d.isValid(); !valid {
	// 	return err
	// }

	if d.DataCallID == 0 {
		sql, boundArgs, _ = sqlBuilder.
			Insert("datacalls").
			Columns("datacall", "deadline").
			Values(d.DataCall, d.Deadline).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", ")).
			ToSql()
	} else {
		sql, boundArgs, _ = sqlBuilder.
			Update("datacalls").
			Set("datacall", d.DataCall).
			Set("deadline", d.Deadline).
			Where("datacallid=?", d.DataCallID).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", ")).
			ToSql()
	}

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return trapError(err)
	}

	err = row.Scan(d.fields()...)

	return trapError(err)
}

func FindDataCalls(ctx context.Context) ([]*DataCall, error) {
	sqlb := sqlBuilder.Select(dataCallColumns...).
		From("datacalls").
		OrderBy("datecreated DESC")

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*DataCall, error) {
		datacall := DataCall{}
		err := row.Scan(datacall.fields()...)
		return &datacall, trapError(err)
	})
}

func FindDataCallByID(ctx context.Context, dataCallID int32) (*DataCall, error) {
	sql, boundArgs, _ := sqlBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid=?", dataCallID).
		ToSql()

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	datacall := DataCall{}
	err = row.Scan(datacall.fields()...)

	return &datacall, err
}

package model

import (
	"context"
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

func (d *DataCall) Save(ctx context.Context) (*DataCall, error) {

	var sqlb SqlBuilder

	// if valid, err := d.isValid(); !valid {
	// 	return err
	// }

	if d.DataCallID == 0 {
		sqlb = stmntBuilder.
			Insert("datacalls").
			Columns("datacall", "deadline").
			Values(d.DataCall, d.Deadline).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", "))
	} else {
		sqlb = stmntBuilder.
			Update("datacalls").
			Set("datacall", d.DataCall).
			Set("deadline", d.Deadline).
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
		OrderBy("deadline DESC", "datacallid DESC")

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
	// find the *previous* datacall by deadline (not datacallid): once historical
	// calls are loaded, a backfilled year can out-id the real prior call, so
	// ordering by datacallid would pick the wrong "previous" for score rollover.
	prevDcSqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid!=?", dataCallID).
		OrderBy("deadline DESC", "datacallid DESC").
		Limit(1)

	return queryRow(context.TODO(), prevDcSqlb, pgx.RowToStructByName[DataCall])
}

func FindLatestDataCall(ctx context.Context) (*DataCall, error) {
	// The current/latest datacall is the one with the furthest-out deadline
	// (datacallid DESC only as a tiebreak): the annual cadence is deadline-driven,
	// and historical loads can carry a higher datacallid than the real current call.
	sqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		OrderBy("deadline DESC", "datacallid DESC").
		Limit(1)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
}

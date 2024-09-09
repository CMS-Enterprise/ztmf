package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type DataCall struct {
	DataCallID  int32   `json:"datacallid"`
	DataCall    string  `json:"datacall"`
	DateCreated float64 `json:"datecreated"`
	Deadline    float64 `json:"deadline"`
}

func FindDataCalls(ctx context.Context) ([]*DataCall, error) {
	sqlb := sqlBuilder.Select("datacallid, datacall, EXTRACT(EPOCH FROM datecreated) as datecreated, EXTRACT(EPOCH FROM deadline) as deadline").
		From("datacalls")

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*DataCall, error) {
		datacall := DataCall{}
		err := rows.Scan(&datacall.DataCallID, &datacall.DataCall, &datacall.DateCreated, &datacall.Deadline)
		return &datacall, trapError(err)
	})
}

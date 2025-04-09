// Package model serves as a lite wrapper around the postgre driver pgx and centralizes
// establishment and management of the lower level db connection. Model methods
// should usually not need to reference the db connection or its methods directly.
package model

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// stmntBuilder is a convenient way to reference a squirrel.StatementBuilder that
// uses the PostgreSQL $1,$2,... format of placeholders
var stmntBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type input struct {
	Include []string `schema:"include"`
	m       map[string]any
}

func (i *input) includes(key string) bool {
	if i.m == nil {
		i.m = map[string]any{}
		for _, s := range i.Include {
			i.m[s] = nil
		}
	}

	_, ok := i.m[key]
	return ok
}

// SqlBuilder allows methods to receive different types like squirrel.InsertBuilder, squirrel.UpdateBuilder, etc. that all implement the ToSql method
type SqlBuilder interface {
	ToSql() (string, []interface{}, error)
}

// query is a proxy to *pgx.Conn.Query and wrapper around pgx.CollectRows, enabling the centralizing of event tracking
func query[T any](ctx context.Context, sqlb SqlBuilder, fn pgx.RowToFunc[T]) ([]T, error) {

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	sql, args, _ := sqlb.ToSql()
	rows, err := conn.Query(ctx, sql, args...)

	if err != nil {
		log.Println(err, sql)
		return nil, trapError(err)
	}

	res, err := pgx.CollectRows(rows, fn)
	if err != nil {
		log.Println(err, sql)
		return nil, trapError(err)
	}

	return res, nil
}

// query is a proxy to *pgx.Conn.Query and wrapper around pgx.CollectOneRow, enabling the centralizing of event tracking
func queryRow[T any](ctx context.Context, sqlb SqlBuilder, fn pgx.RowToFunc[T]) (*T, error) {

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	sql, args, _ := sqlb.ToSql()

	rows, err := conn.Query(ctx, sql, args...)

	if err != nil {
		log.Println(err, sql)
		return nil, trapError(err)
	}

	res, err := pgx.CollectOneRow(rows, fn)
	if err != nil {
		return nil, trapError(err)
	}

	recordEvent(ctx, sqlb, res)

	return &res, nil
}

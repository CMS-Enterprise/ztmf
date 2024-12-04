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

// SqlBuilder allows methods to receive different types like squirrel.InsertBuilder, squirrel.UpdateBuilder, etc. that all implement the ToSql method
type sqlBuilder interface {
	ToSql() (string, []interface{}, error)
}

// query is a proxy to *pgx.Conn.Query
func query(ctx context.Context, sqlb sqlBuilder) (pgx.Rows, error) {

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	sql, args, _ := sqlb.ToSql()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		log.Println(err, sql)
	}
	return rows, err
}

// queryRow is a proxy to *pgx.Conn.QueryRow
func queryRow(ctx context.Context, sqlb sqlBuilder) (pgx.Row, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	sql, args, _ := sqlb.ToSql()

	row := conn.QueryRow(ctx, sql, args...)
	return row, nil
}

// exec is a proxy to *pgx.Conn.Exec
func exec(ctx context.Context, sqlb sqlBuilder) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}

	sql, args, _ := sqlb.ToSql()

	_, err = conn.Exec(ctx, sql, args...)
	if err != nil {
		log.Println(err, sql)
	}
	return err
}

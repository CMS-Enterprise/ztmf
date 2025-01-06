// Package model serves as a lite wrapper around the postgre driver pgx and centralizes
// establishment and management of the lower level db connection. Model methods
// should usually not need to reference the db connection or its methods directly.
package model

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// sqlBuilder is a convenient way to reference a squirrel.StatementBuilder that
// uses the PostgreSQL $1,$2,... format of placeholders
var sqlBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// query is a proxy to *pgx.Conn.Query
func query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	return conn.Query(ctx, sql, args...)
}

// queryRow is a proxy to *pgx.Conn.QueryRow
func queryRow(ctx context.Context, sql string, args ...any) (pgx.Row, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	row := conn.QueryRow(ctx, sql, args...)
	return row, nil
}

// exec is a proxy to *pgx.Conn.Exec
func exec(ctx context.Context, sql string, args ...any) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, sql, args...)
	return err
}

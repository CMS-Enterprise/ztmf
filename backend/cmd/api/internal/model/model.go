// model serves as a lite wrapper around the postgre driver pgx and centralizes getting the db connection
// to reduce repetitive code in table specific functions
package model

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Query is a proxy to *pgx.Conn.Query
func Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return conn.Query(ctx, sql, args...)
}

// QueryRow is a proxy to *pgx.Conn.QueryRow
func QueryRow(ctx context.Context, sql string, args ...any) (pgx.Row, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	row := conn.QueryRow(ctx, sql, args...)
	return row, nil
}

// Exec is a proxy to *pgx.Conn.Exec
func Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return pgconn.NewCommandTag(""), err
	}

	return conn.Exec(ctx, sql, args...)
}

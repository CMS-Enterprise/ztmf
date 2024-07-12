// model serves as a lite wrapper around the postgre driver pgx and centralizes getting the db connection
// to reduce repetitive code in table specific functions
package model

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// query is a proxy to *pgx.Conn.Query
func query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return conn.Query(ctx, sql, args...)
}

// queryRow is a proxy to *pgx.Conn.QueryRow
func queryRow(ctx context.Context, sql string, args ...any) (pgx.Row, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	row := conn.QueryRow(ctx, sql, args...)
	return row, nil
}

// exec is a proxy to *pgx.Conn.Exec
// func exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
// 	conn, err := db.Conn(ctx)
// 	if err != nil {
// 		return pgconn.NewCommandTag(""), err
// 	}

// 	return conn.Exec(ctx, sql, args...)
// }

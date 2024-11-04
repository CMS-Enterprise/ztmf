package migrations

import (
	"context"
	"os"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
)

func populate(path string) error {
	sql, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ctx := context.TODO()

	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, string(sql))
	return err
}

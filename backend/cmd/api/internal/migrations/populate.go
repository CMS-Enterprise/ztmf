package migrations

import (
	"context"
	"log"
	"os"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
)

func populate(path string) error {
	log.Printf("Populating database with %s\n", path)
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

package migrations

import (
	"context"
	"log"
	"os"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
)

func populate(path string) error {
	ctx := context.TODO()

	// The seed script is not idempotent (it INSERTs fixed-key rows and fails with a
	// duplicate-key error on a second run). A local dev volume persists across restarts,
	// so on every `make dev-up` after the first the seed would otherwise re-run against
	// an already-populated database and crash the api on boot. Skip seeding when the
	// database already holds data, preserving whatever the developer has loaded (e.g. a
	// dev/prod DB sync). Ephemeral test databases (Emberfall E2E) start empty each run,
	// so they still seed normally.
	seeded, err := alreadySeeded(ctx)
	if err != nil {
		return err
	}
	if seeded {
		log.Print("Skipping populate: database already contains data")
		return nil
	}

	log.Printf("Populating database with %s", path)
	sql, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, string(sql))
	return err
}

// alreadySeeded reports whether the database already holds seed data. It checks
// fismasystems, a core table created by migrations and populated by the seed script.
func alreadySeeded(ctx context.Context) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM fismasystems").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

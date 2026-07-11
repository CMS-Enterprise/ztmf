package migrations

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/require"
)

// TestPopulateSkipsWhenSeededIntegration pins the seed gate against a real
// database: once data exists, populate() must be a no-op rather than re-running
// the non-idempotent seed script and crashing the api on boot with a
// duplicate-key error. make test-integration seeds the ephemeral DB on startup,
// so fismasystems already holds rows by the time this runs.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestPopulateSkipsWhenSeededIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	conn.Release()

	seeded, err := alreadySeeded(ctx)
	require.NoError(t, err)
	require.True(t, seeded, "integration DB is seeded on startup, so alreadySeeded must report true")

	// The path is deliberately nonexistent: if the gate works, populate() returns
	// before ever reading the file. A missing-file error here would mean the gate
	// let execution fall through to os.ReadFile on an already-seeded database.
	err = populate("/does-not-exist-populate-gate-test.sql")
	require.NoError(t, err, "populate must skip (without touching the filesystem) when the database already has data")
}

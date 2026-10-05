package migrations

import (
	"context"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLastUpdatedAtBackfillIntegration pins 0063: last_updated_at takes the
// newest created/updated event and ignores imported provenance, matching the
// progress lateral it replaced. tern ran 0063 against empty tables, so the SQL
// is re-run here on fixtures inside a rolled-back transaction.
func TestLastUpdatedAtBackfillIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	up, down := migrationSQL(t, "backfill scores.last_updated_at")

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	dataCallID, _, _, optA, _ := dirtyScoresFixture(t, ctx, tx)

	var userID string
	require.NoError(t, tx.QueryRow(ctx, `SELECT userid FROM users LIMIT 1`).Scan(&userID))

	rows, err := tx.Query(ctx, `SELECT fismasystemid FROM fismasystems ORDER BY fismasystemid LIMIT 3`)
	require.NoError(t, err)
	var systems []int32
	for rows.Next() {
		var id int32
		require.NoError(t, rows.Scan(&id))
		systems = append(systems, id)
	}
	require.NoError(t, rows.Err())
	require.Len(t, systems, 3)

	insertScore := func(systemID int32) int32 {
		var id int32
		require.NoError(t, tx.QueryRow(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid)
			VALUES ($1, $2, $3) RETURNING scoreid
		`, systemID, optA, dataCallID).Scan(&id))
		return id
	}
	insertEvent := func(scoreID int32, action string, at time.Time) {
		_, err := tx.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, $2, 'public.scores', $3, jsonb_build_object('scoreid', $4::int))
		`, userID, action, at, scoreID)
		require.NoError(t, err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	edited, imported, untouched := insertScore(systems[0]), insertScore(systems[1]), insertScore(systems[2])
	insertEvent(edited, "created", base)
	insertEvent(edited, "updated", base.Add(time.Hour))
	insertEvent(edited, "imported", base.Add(2*time.Hour))
	insertEvent(imported, "imported", base)

	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)

	lastUpdated := func(scoreID int32) *time.Time {
		var at *time.Time
		require.NoError(t, tx.QueryRow(ctx, `SELECT last_updated_at FROM scores WHERE scoreid = $1`, scoreID).Scan(&at))
		return at
	}
	if at := lastUpdated(edited); assert.NotNil(t, at, "an edited row must be backfilled") {
		assert.True(t, at.Equal(base.Add(time.Hour)), "the newest created/updated event wins; a later import must not")
	}
	assert.Nil(t, lastUpdated(imported), "imported-only provenance must stay NULL")
	assert.Nil(t, lastUpdated(untouched), "a row with no events must stay NULL")

	_, err = tx.Exec(ctx, down)
	require.NoError(t, err)
	assert.Nil(t, lastUpdated(edited), "down must clear the backfill")
}

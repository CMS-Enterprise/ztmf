package migrations

import (
	"context"
	"errors"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctionOptionsScoreCheckIntegration pins migration 0067: the database
// rejects an option score outside 1..4 even when the app-level validation is
// bypassed, accepts both ends of the scale, and the up migration is a no-op on
// retry. Everything runs inside a transaction that is always rolled back.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.
func TestFunctionOptionsScoreCheckIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	up, down := migrationSQL(t, "constrain functionoptions.score")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var functionID int32
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT functionid FROM functions ORDER BY functionid LIMIT 1`).Scan(&functionID))

	// Each insert runs in a savepoint so a rejected one does not abort the
	// outer transaction.
	insert := func(score int) error {
		sp, err := tx.Begin(ctx)
		require.NoError(t, err)
		_, err = sp.Exec(ctx, `
			INSERT INTO functionoptions (functionid, score, optionname, description)
			VALUES ($1, $2, 'score check fixture', '')
		`, functionID, score)
		if err != nil {
			require.NoError(t, sp.Rollback(ctx))
			return err
		}
		return sp.Commit(ctx)
	}

	requireCheckViolation := func(t *testing.T, err error) {
		t.Helper()
		var pgErr *pgconn.PgError
		require.True(t, errors.As(err, &pgErr), "expected a Postgres error, got %v", err)
		assert.Equal(t, "23514", pgErr.Code, "expected check_violation")
		assert.Equal(t, "functionoptions_score_check", pgErr.ConstraintName)
	}

	for _, score := range []int{0, 5, -1} {
		requireCheckViolation(t, insert(score))
	}
	for _, score := range []int{1, 4} {
		assert.NoError(t, insert(score), "score %d is on the scale", score)
	}

	// Running up again must not fail on the constraint that already exists.
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err, "0067 up must be a no-op when the constraint exists")

	// down removes it, and up puts it back.
	_, err = tx.Exec(ctx, down)
	require.NoError(t, err)
	assert.NoError(t, insert(5), "with the constraint dropped, an off-scale score is accepted")
	assert.Equal(t, 0, scoreCheckCount(t, ctx, tx), "down must drop the constraint")

	// up validates existing rows, so the off-scale row above must make it fail.
	sp, err := tx.Begin(ctx)
	require.NoError(t, err)
	_, err = sp.Exec(ctx, up)
	requireCheckViolation(t, err)
	require.NoError(t, sp.Rollback(ctx))

	_, err = tx.Exec(ctx, `DELETE FROM functionoptions WHERE optionname = 'score check fixture' AND score = 5`)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)
	assert.Equal(t, 1, scoreCheckCount(t, ctx, tx), "up must restore the constraint")
}

func scoreCheckCount(t *testing.T, ctx context.Context, tx pgx.Tx) int {
	t.Helper()
	var n int
	require.NoError(t, tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_constraint
		WHERE conname = 'functionoptions_score_check'
		  AND conrelid = 'public.functionoptions'::regclass
	`).Scan(&n))
	return n
}

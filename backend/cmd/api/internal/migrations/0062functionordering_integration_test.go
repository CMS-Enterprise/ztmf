package migrations

import (
	"context"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctionOrderingMigrationIntegration runs migration 0062's real SQL against
// a live database, chained onto 0056's.
//
// Same constraint as TestOrderingDataMigrationIntegration: tern applies
// migrations before the seed loads, so on an ephemeral empire-seeded database
// both migrations execute against empty catalog tables and write nothing. The
// test pulls both migrations' SQL out of the registry and runs them in order
// inside a transaction it always rolls back, on canonically-named fixture rows
// it inserts itself.
//
// Chaining 0056 rather than hand-setting questions.ordr is the point: 0062 copies
// whatever rank 0056 assigned, so running them together is what proves a
// canonical function name reaches functions.ordr.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.
func TestFunctionOrderingMigrationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	var orderingUp, up, down string
	for _, m := range registry {
		switch {
		case strings.Contains(m.name, "pillars.ordr and questions.ordr"):
			orderingUp = m.upSQL
		case strings.Contains(m.name, "functions.ordr"):
			up, down = m.upSQL, m.downSQL
		}
	}
	require.NotEmpty(t, orderingUp, "migration 0056 not found in the registry; was it renamed?")
	require.NotEmpty(t, up, "migration 0062 not found in the registry; was it renamed?")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	// Everything below is scratch: the rollback is the cleanup.
	defer func() { _ = tx.Rollback(ctx) }()

	var identityPillarID int32
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT pillarid FROM pillars WHERE pillar = 'Identity'`).Scan(&identityPillarID),
		"catalog must contain the Identity pillar")

	// Pre-state: unranked, which is what 0056 expects to find.
	_, err = tx.Exec(ctx, `UPDATE pillars SET ordr = 0`)
	require.NoError(t, err)

	// A canonically-named question, an empire-seed-style one 0056 cannot rank,
	// and one whose function already carries a deliberate rank.
	canonical := insertQuestion(ctx, t, tx, identityPillarID, 0, "AccessManagement")
	unmatched := insertQuestion(ctx, t, tx, identityPillarID, 0, "Imperial Identity Verification (fixture)")
	preranked := insertQuestion(ctx, t, tx, identityPillarID, 0, "RiskAssessment")
	_, err = tx.Exec(ctx,
		`UPDATE functions SET ordr = 42 WHERE questionid = $1`, preranked)
	require.NoError(t, err)

	// A second edition of the canonical question, the shape that makes the
	// name-keyed rank worth copying: both rows must end up ranked. Its rank is
	// NULL rather than 0 because ordr is nullable, and a bare `ordr = 0` guard
	// is NULL - not true - on that row, which would skip it silently.
	_, err = tx.Exec(ctx, `
		INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid, ordr)
		VALUES ('AccessManagement', 'fixture second edition', 'fixture2', $1, $2, NULL)
	`, canonical, identityPillarID)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, orderingUp)
	require.NoError(t, err, "migration 0056 up SQL must apply cleanly")
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err, "migration up SQL must apply cleanly")

	assert.Equal(t, []int{104, 104}, functionOrdrs(ctx, t, tx, canonical),
		"every edition of a canonically-named question takes its question's rank")
	assert.Equal(t, []int{0}, functionOrdrs(ctx, t, tx, unmatched),
		"a function whose question 0056 could not rank stays unranked")
	assert.Equal(t, []int{42}, functionOrdrs(ctx, t, tx, preranked),
		"a function already carrying a rank is left alone")

	// Idempotent: a second run is a no-op rather than a reshuffle.
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)
	assert.Equal(t, []int{104, 104}, functionOrdrs(ctx, t, tx, canonical),
		"re-running the migration must not change the ranks it already set")

	_, err = tx.Exec(ctx, down)
	require.NoError(t, err, "migration down SQL must apply cleanly")

	assert.Equal(t, []int{0, 0}, functionOrdrs(ctx, t, tx, canonical), "down must clear the ranks it set")
	assert.Equal(t, []int{0}, functionOrdrs(ctx, t, tx, preranked), "down clears every function rank")
}

// functionOrdrs returns the ranks of a question's functions, ordered so the
// assertion does not depend on insertion order. A NULL rank surfaces as -1
// rather than a scan error, so a row the migration skipped reads as a value in
// the diff instead of failing somewhere unrelated.
func functionOrdrs(ctx context.Context, t *testing.T, tx pgx.Tx, questionID int32) []int {
	t.Helper()

	rows, err := tx.Query(ctx,
		`SELECT coalesce(ordr, -1) FROM functions WHERE questionid = $1 ORDER BY functionid`, questionID)
	require.NoError(t, err)
	defer rows.Close()

	var ordrs []int
	for rows.Next() {
		var ordr int
		require.NoError(t, rows.Scan(&ordr))
		ordrs = append(ordrs, ordr)
	}
	require.NoError(t, rows.Err())
	return ordrs
}

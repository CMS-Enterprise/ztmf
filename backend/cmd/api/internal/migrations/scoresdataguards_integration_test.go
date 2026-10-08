package migrations

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The ztmf#491 migrations refuse to run against dirty data, and that refusal is
// the entire merge gate: the duplicate repair is manual and out-of-band, so the
// only thing standing between "someone merged early" and a silently wrong
// database is these two guards. A guard nobody tests is a guard that quietly
// stops working, so each one is exercised here against data the test dirties
// itself, inside a transaction it always rolls back.
//
// Both migrations have already run by the time this executes (tern applies them
// at startup), so each test first undoes the relevant piece - drops the index,
// or manufactures an orphan - to recreate the pre-migration state.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.

func migrationSQL(t *testing.T, nameFragment string) (string, string) {
	t.Helper()
	for _, m := range registry {
		if strings.Contains(m.name, nameFragment) {
			return m.upSQL, m.downSQL
		}
	}
	t.Fatalf("no migration matching %q in the registry; was it renamed?", nameFragment)
	return "", ""
}

// dirtyScoresFixture inserts a data call and returns it along with a system and a
// function carrying two selectable options, so a second answer to the SAME
// question can be manufactured. Transaction-scoped; the rollback is the cleanup.
func dirtyScoresFixture(t *testing.T, ctx context.Context, tx pgx.Tx) (dataCallID, systemID, functionID, optA, optB int32) {
	t.Helper()

	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("scores_guard_%d", 1)).Scan(&dataCallID))

	require.NoError(t, tx.QueryRow(ctx, `
		SELECT fo.functionid, MIN(fo.functionoptionid), MAX(fo.functionoptionid)
		  FROM functionoptions fo
		 GROUP BY fo.functionid
		HAVING COUNT(*) >= 2
		 ORDER BY fo.functionid
		 LIMIT 1
	`).Scan(&functionID, &optA, &optB))

	require.NoError(t, tx.QueryRow(ctx,
		`SELECT fismasystemid FROM fismasystems ORDER BY fismasystemid LIMIT 1`).Scan(&systemID))

	return
}

// TestUniqueAnswerMigrationRefusesDuplicatesIntegration pins migration 0061's
// ZTMF491_DUPLICATE_ANSWERS guard.
//
// Without it a premature deploy fails on a bare 23505 from inside a crash-looping
// ECS task, which says nothing about what to do. The assertion is on the token and
// the ticket reference, because those are what the person reading CloudWatch at
// the time actually needs.
func TestUniqueAnswerMigrationRefusesDuplicatesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	up, _ := migrationSQL(t, "one answer per system per question per data call")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	dataCallID, systemID, _, optA, optB := dirtyScoresFixture(t, ctx, tx)

	// Recreate the pre-migration state. DDL is transactional in Postgres, so the
	// rollback restores the index with no cleanup of our own.
	_, err = tx.Exec(ctx, `DROP INDEX public.scores_fismasystem_datacall_function_uniq`)
	require.NoError(t, err, "migration 0061's index must exist before this test can remove it")

	for _, opt := range []int32{optA, optB} {
		_, err = tx.Exec(ctx, `
			INSERT INTO scores (fismasystemid, functionoptionid, datacallid) VALUES ($1, $2, $3)
		`, systemID, opt, dataCallID)
		require.NoError(t, err)
	}

	_, err = tx.Exec(ctx, up)
	require.Error(t, err, "0061 must refuse to build the index while a duplicate exists")
	assert.Contains(t, err.Error(), "ZTMF491_DUPLICATE_ANSWERS",
		"the failure must carry the greppable token, not a bare unique-violation")
	assert.Contains(t, err.Error(), "ztmf#491",
		"the failure must name the ticket that explains it")
}

// TestUniqueAnswerMigrationSucceedsWhenCleanIntegration is the other half: the
// guard must not be a permanent blocker. Without this, a guard that always threw
// would pass the test above and still be broken.
func TestUniqueAnswerMigrationSucceedsWhenCleanIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	up, _ := migrationSQL(t, "one answer per system per question per data call")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DROP INDEX public.scores_fismasystem_datacall_function_uniq`)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, up)
	require.NoError(t, err, "0061 must apply cleanly against a database with no duplicates")

	var exists bool
	require.NoError(t, tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_indexes
		                WHERE schemaname = 'public'
		                  AND indexname = 'scores_fismasystem_datacall_function_uniq')
	`).Scan(&exists))
	assert.True(t, exists, "the index must exist after a clean apply")
}

// TestFunctionIDMigrationRefusesOrphansIntegration pins migration 0060's
// ZTMF491_ORPHANED_SCORES guard.
//
// The orphan is manufactured under session_replication_role = 'replica', which
// is not a trick for the test's convenience - it is the exact mechanism that
// produced the orphans in the first place. Bulk loads run in replica mode, which
// disables triggers and FK checks, so a score naming a functionoptionid that
// does not exist gets in despite the FK. NOT NULL is a column constraint rather
// than a trigger, so functionid must still be supplied explicitly.
//
// Skips rather than fails if the role cannot be set: it needs superuser, which
// the ephemeral test container grants but a locked-down database may not.
func TestFunctionIDMigrationRefusesOrphansIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	up, _ := migrationSQL(t, "denormalize functionid onto scores")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	dataCallID, systemID, functionID, _, _ := dirtyScoresFixture(t, ctx, tx)

	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
		t.Skipf("cannot set session_replication_role (needs superuser): %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, functionid)
		VALUES ($1, -1, $2, $3)
	`, systemID, dataCallID, functionID)
	require.NoError(t, err, "replica mode must bypass the FK and the trigger - that is the premise of this test")

	_, err = tx.Exec(ctx, `SET LOCAL session_replication_role = 'origin'`)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, up)
	require.Error(t, err, "0060 must refuse to run while a score names a nonexistent functionoption")
	assert.Contains(t, err.Error(), "ZTMF491_ORPHANED_SCORES",
		"the failure must carry the greppable token, not a bare not-null violation")
	assert.Contains(t, err.Error(), "ztmf#491",
		"the failure must name the ticket that explains it")
}

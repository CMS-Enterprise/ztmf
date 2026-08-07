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

// TestOrderingDataMigrationIntegration runs migration 0056's real SQL against a
// live database and checks what it did.
//
// It cannot rely on the migration having already run: tern applies migrations
// before the seed loads, so on an ephemeral empire-seeded database 0056 executes
// against empty catalog tables and writes nothing. Rather than skip, the test
// pulls the migration's own up/down SQL out of the registry and executes it
// inside a transaction it always rolls back, on fixture rows it inserts itself.
// That covers the three behaviors the migration promises regardless of which
// fixture the database carries: canonical names get ranked, unrecognized names
// are left at 0, and the legacy 901-907 band is never touched.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.
func TestOrderingDataMigrationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	var up, down string
	for _, m := range registry {
		if strings.Contains(m.name, "pillars.ordr and questions.ordr") {
			up, down = m.upSQL, m.downSQL
			break
		}
	}
	require.NotEmpty(t, up, "migration 0056 not found in the registry; was it renamed?")

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

	// Pre-state: unranked, which is what the migration expects to find.
	_, err = tx.Exec(ctx, `UPDATE pillars SET ordr = 0`)
	require.NoError(t, err)

	// Three fixture questions in one pillar: one whose function name is in the
	// canonical map, one whose name is not (the empire-seed case), and one
	// already sitting in the legacy band.
	canonical := insertQuestion(ctx, t, tx, identityPillarID, 0, "Authentication-Users")
	unmatched := insertQuestion(ctx, t, tx, identityPillarID, 0, "Imperial Identity Verification (fixture)")
	legacy := insertQuestion(ctx, t, tx, identityPillarID, 905, "Future Plan Fixture")

	_, err = tx.Exec(ctx, up)
	require.NoError(t, err, "migration up SQL must apply cleanly")

	assert.Equal(t,
		[]string{"Identity", "Devices", "Networks", "Applications", "Data", "CrossCutting"},
		pillarNamesByRank(ctx, t, tx),
		"pillars must be ranked in the CISA ZTMM sequence the questionnaire renders")

	assert.EqualValues(t, 101, questionOrdr(ctx, t, tx, canonical),
		"a question whose function name is in the map takes pillar_rank*100 + function_index")
	assert.EqualValues(t, 0, questionOrdr(ctx, t, tx, unmatched),
		"a question whose function name is not in the map stays unranked")
	assert.EqualValues(t, 905, questionOrdr(ctx, t, tx, legacy),
		"the legacy 901-907 band must survive the migration untouched")

	_, err = tx.Exec(ctx, down)
	require.NoError(t, err, "migration down SQL must apply cleanly")

	var rankedPillars int
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT count(*) FROM pillars WHERE ordr <> 0`).Scan(&rankedPillars))
	assert.Zero(t, rankedPillars, "down must clear every pillar rank")

	assert.EqualValues(t, 0, questionOrdr(ctx, t, tx, canonical), "down must clear the ranks it set")
	assert.EqualValues(t, 905, questionOrdr(ctx, t, tx, legacy),
		"down must preserve the legacy band it never wrote")
}

// insertQuestion adds a question with the given rank plus one function naming
// it, mirroring the questions -> functions shape the migration keys on.
func insertQuestion(ctx context.Context, t *testing.T, tx pgx.Tx, pillarID int32, ordr int, functionName string) int32 {
	t.Helper()

	var questionID int32
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO questions (question, notesprompt, pillarid, ordr)
		VALUES ($1, 'fixture', $2, $3)
		RETURNING questionid
	`, "ordering fixture: "+functionName, pillarID, ordr).Scan(&questionID))

	var functionID int32
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
		VALUES ($1, 'fixture', 'fixture', $2, $3)
		RETURNING functionid
	`, functionName, questionID, pillarID).Scan(&functionID))

	return questionID
}

func questionOrdr(ctx context.Context, t *testing.T, tx pgx.Tx, questionID int32) int {
	t.Helper()

	var ordr int
	require.NoError(t, tx.QueryRow(ctx,
		`SELECT ordr FROM questions WHERE questionid = $1`, questionID).Scan(&ordr))
	return ordr
}

func pillarNamesByRank(ctx context.Context, t *testing.T, tx pgx.Tx) []string {
	t.Helper()

	rows, err := tx.Query(ctx, `SELECT pillar FROM pillars ORDER BY ordr`)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

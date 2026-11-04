package migrations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataCallsFiscalYearIntegration pins migration 0070: the backfill reads
// the fiscal year and cadence out of each existing name, fails and lists the
// rows it cannot read instead of guessing, is a no-op on retry, and the CHECKs
// hold afterwards. tern ran 0070 against the seed's data calls already, so the
// test runs down, loads its own names, and runs up again, all inside a
// transaction that is always rolled back.
//
// Requires DB_* env vars pointing at a migrated ZTMF database. Skipped under
// `go test -short`.
func TestDataCallsFiscalYearIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	up, down := migrationSQL(t, "store fiscal_year and cadence on datacalls")

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, down)
	require.NoError(t, err)
	assert.Equal(t, 0, fiscalYearColumnCount(t, ctx, tx), "down must drop both columns")

	// The seed carries names the backfill cannot read (it inserts after
	// migrations and sets the columns itself), so give every existing row a
	// parseable name. Only the fixtures below are under test.
	_, err = tx.Exec(ctx, `UPDATE datacalls SET datacall = 'FY26 seed ' || datacallid`)
	require.NoError(t, err)

	insert := func(name string) int32 {
		var id int32
		require.NoError(t, tx.QueryRow(ctx, `
			INSERT INTO datacalls (datacall, datecreated, deadline)
			VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz)
			RETURNING datacallid
		`, name).Scan(&id))
		return id
	}

	type want struct {
		year    int32
		cadence string
	}
	cases := map[string]want{
		"FY24 Q1":                     {2024, "quarterly"},
		"FY2025 Q3":                   {2025, "quarterly"},
		"FY25 ZTM":                    {2025, "annual"},
		"FY2026 ZTM":                  {2026, "annual"},
		"FY2027 Zero Trust Data Call": {2027, "annual"},
		"  fy23 q4  ":                 {2023, "quarterly"},
		"FY26 HQ1 Review":             {2026, "annual"},
	}
	ids := make(map[string]int32, len(cases))
	for name := range cases {
		ids[name] = insert(name)
	}

	// Names with no usable year make up fail, and the error lists each one.
	sp, err := tx.Begin(ctx)
	require.NoError(t, err)
	var bad []string
	for _, name := range []string{"Zero Trust Data Call", "FY202 Review", "FY1999 Legacy", "2026 FY ZTM"} {
		var id int32
		require.NoError(t, sp.QueryRow(ctx, `
			INSERT INTO datacalls (datacall, datecreated, deadline)
			VALUES ($1, NOW(), '2100-01-01T00:00:00Z'::timestamptz)
			RETURNING datacallid
		`, name).Scan(&id))
		bad = append(bad, fmt.Sprintf("%d '%s'", id, name))
	}
	_, err = sp.Exec(ctx, up)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "expected a Postgres error from up, got %v", err)
	assert.Contains(t, pgErr.Message, "DATACALL_FISCAL_YEAR_BACKFILL")
	for _, b := range bad {
		assert.Contains(t, pgErr.Message, b, "the error must list every unparseable row")
	}
	for name, id := range ids {
		assert.NotContains(t, pgErr.Message, fmt.Sprintf("%d '%s'", id, name), "a parseable row must not be listed")
	}
	require.NoError(t, sp.Rollback(ctx))

	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)
	assert.Equal(t, 2, fiscalYearColumnCount(t, ctx, tx))

	for name, w := range cases {
		var year int32
		var cadence string
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT fiscal_year, cadence FROM datacalls WHERE datacallid = $1`, ids[name]).Scan(&year, &cadence))
		assert.Equal(t, w.year, year, "fiscal_year for %q", name)
		assert.Equal(t, w.cadence, cadence, "cadence for %q", name)
	}

	// A retry leaves the backfilled values alone, even on a row whose name no
	// longer parses.
	_, err = tx.Exec(ctx, `UPDATE datacalls SET datacall = 'Renamed after backfill' WHERE datacallid = $1`, ids["FY24 Q1"])
	require.NoError(t, err)
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err, "up must be a no-op once the columns are filled")
	var year int32
	require.NoError(t, tx.QueryRow(ctx, `SELECT fiscal_year FROM datacalls WHERE datacallid = $1`, ids["FY24 Q1"]).Scan(&year))
	assert.Equal(t, int32(2024), year)

	// The constraints hold for writes that skip the API.
	write := func(sql string, args ...any) error {
		sp, err := tx.Begin(ctx)
		require.NoError(t, err)
		_, err = sp.Exec(ctx, sql, args...)
		require.NoError(t, sp.Rollback(ctx))
		return err
	}
	requireViolation := func(t *testing.T, err error, code, constraint string) {
		t.Helper()
		var pgErr *pgconn.PgError
		require.True(t, errors.As(err, &pgErr), "expected a Postgres error, got %v", err)
		assert.Equal(t, code, pgErr.Code)
		if constraint != "" {
			assert.Equal(t, constraint, pgErr.ConstraintName)
		}
	}
	requireViolation(t, write(`INSERT INTO datacalls (datacall, deadline) VALUES ('no year', NOW())`), "23502", "")
	requireViolation(t, write(`INSERT INTO datacalls (datacall, deadline, fiscal_year) VALUES ('two digit', NOW(), 26)`), "23514", "datacalls_fiscal_year_check")
	requireViolation(t, write(`INSERT INTO datacalls (datacall, deadline, fiscal_year, cadence) VALUES ('monthly', NOW(), 2026, 'monthly')`), "23514", "datacalls_cadence_check")

	var cadence string
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, deadline, fiscal_year) VALUES ('default cadence', NOW(), 2026)
		RETURNING cadence
	`).Scan(&cadence))
	assert.Equal(t, "annual", cadence, "cadence defaults to annual")

	_, err = tx.Exec(ctx, down)
	require.NoError(t, err)
	assert.Equal(t, 0, fiscalYearColumnCount(t, ctx, tx), "down must drop both columns")
}

func fiscalYearColumnCount(t *testing.T, ctx context.Context, tx pgx.Tx) int {
	t.Helper()
	var n int
	require.NoError(t, tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'datacalls'
		  AND column_name IN ('fiscal_year', 'cadence')
	`).Scan(&n))
	return n
}

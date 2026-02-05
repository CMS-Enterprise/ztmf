package cfacts

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncResult contains the outcome of a CFACTS sync operation.
type SyncResult struct {
	RowsInserted int64
	Duration     time.Duration
}

// SyncToPostgres truncates cfacts_systems and inserts all systems in a single transaction.
// If dryRun is true, the transaction is rolled back after validation.
func SyncToPostgres(ctx context.Context, pool *pgxpool.Pool, systems []CfactsSystem, dryRun bool) (*SyncResult, error) {
	startTime := time.Now()

	if len(systems) == 0 {
		return &SyncResult{Duration: time.Since(startTime)}, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is no-op after commit

	// Truncate table
	if _, err := tx.Exec(ctx, "TRUNCATE TABLE cfacts_systems"); err != nil {
		return nil, fmt.Errorf("failed to truncate cfacts_systems: %w", err)
	}

	log.Printf("Truncated cfacts_systems, inserting %d rows (dryRun=%t)", len(systems), dryRun)

	// Build batch INSERT with parameterized queries
	const batchSize = 100
	totalInserted := int64(0)

	for i := 0; i < len(systems); i += batchSize {
		end := i + batchSize
		if end > len(systems) {
			end = len(systems)
		}
		batch := systems[i:end]

		inserted, err := insertBatch(ctx, tx, batch, i+1)
		if err != nil {
			return nil, err
		}
		totalInserted += inserted

		if (i+batchSize)%1000 == 0 || end == len(systems) {
			log.Printf("Inserted %d/%d rows into cfacts_systems...", totalInserted, len(systems))
		}
	}

	if dryRun {
		log.Printf("DRY RUN: Rolling back %d rows", totalInserted)
		if err := tx.Rollback(ctx); err != nil {
			return nil, fmt.Errorf("failed to rollback dry-run transaction: %w", err)
		}
	} else {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
		log.Printf("Committed %d rows to cfacts_systems", totalInserted)
	}

	return &SyncResult{
		RowsInserted: totalInserted,
		Duration:     time.Since(startTime),
	}, nil
}

// insertBatch inserts a batch of systems using a single multi-row INSERT.
func insertBatch(ctx context.Context, tx pgx.Tx, batch []CfactsSystem, startRow int) (int64, error) {
	// Build: INSERT INTO cfacts_systems (col1, ..., col15, synced_at) VALUES ($1,...,$15, NOW()), ($16,...,$30, NOW()), ...
	numCols := len(cfactsColumns)
	valueSets := make([]string, len(batch))
	args := make([]any, 0, len(batch)*numCols)

	for i, sys := range batch {
		placeholders := make([]string, numCols)
		for j := range cfactsColumns {
			paramNum := i*numCols + j + 1
			placeholders[j] = fmt.Sprintf("$%d", paramNum)
		}
		valueSets[i] = fmt.Sprintf("(%s, NOW())", strings.Join(placeholders, ", "))
		args = append(args, sys.values()...)
	}

	query := fmt.Sprintf("INSERT INTO cfacts_systems (%s, synced_at) VALUES %s",
		strings.Join(cfactsColumns, ", "),
		strings.Join(valueSets, ", "))

	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert batch starting at row %d: %w", startRow, err)
	}

	return tag.RowsAffected(), nil
}

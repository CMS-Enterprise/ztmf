package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/export"
)

// Synchronizer handles the data synchronization between PostgreSQL and Snowflake
type Synchronizer struct {
	dryRun     bool
	pgClient   *export.PostgresClient
	snowClient *export.SnowflakeClient
}

// SyncOptions configures how the sync should be performed
type SyncOptions struct {
	Tables      []string // If empty, sync all tables
	FullRefresh bool     // If true, truncate and reload
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	StartTime    time.Time
	EndTime      time.Time
	TablesSync   []TableSyncResult
	TotalRows    int64
	TotalErrors  int
	DryRun       bool
}

// TableSyncResult contains sync results for a single table
type TableSyncResult struct {
	PostgresTable  string
	SnowflakeTable string
	RowsExtracted  int64
	RowsLoaded     int64
	Duration       time.Duration
	Error          error
}

// NewSynchronizer creates a new data synchronizer
func NewSynchronizer(ctx context.Context, dryRun bool) (*Synchronizer, error) {
	log.Printf("Initializing synchronizer - DryRun: %t", dryRun)
	
	sync := &Synchronizer{
		dryRun: dryRun,
	}
	
	// Initialize PostgreSQL connection
	if !dryRun {
		pgClient, err := export.NewPostgresClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL client: %w", err)
		}
		sync.pgClient = pgClient
		
		// Initialize Snowflake connection
		snowClient, err := export.NewSnowflakeClient(ctx)
		if err != nil {
			pgClient.Close() // Clean up PG connection on failure
			return nil, fmt.Errorf("failed to initialize Snowflake client: %w", err)
		}
		sync.snowClient = snowClient
	}
	
	return sync, nil
}

// Close cleans up resources
func (s *Synchronizer) Close() {
	log.Println("Closing synchronizer connections...")
	
	if s.pgClient != nil {
		s.pgClient.Close()
	}
	
	if s.snowClient != nil {
		s.snowClient.Close()
	}
}

// ExecuteSync performs the data synchronization
func (s *Synchronizer) ExecuteSync(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	startTime := time.Now()
	
	log.Printf("Starting sync execution - DryRun: %t", s.dryRun)
	
	result := &SyncResult{
		StartTime: startTime,
		DryRun:    s.dryRun,
	}
	
	// Get tables to sync
	tablesToSync, err := s.getTablesToSync(opts.Tables)
	if err != nil {
		return nil, fmt.Errorf("failed to determine tables to sync: %w", err)
	}
	
	log.Printf("Tables to sync: %v", tablesToSync)
	
	// Process each table
	for _, table := range tablesToSync {
		tableResult := s.syncTable(ctx, table, opts.FullRefresh)
		result.TablesSync = append(result.TablesSync, tableResult)
		
		if tableResult.Error != nil {
			result.TotalErrors++
			log.Printf("Error syncing table %s: %v", table.PostgresTable, tableResult.Error)
		} else {
			result.TotalRows += tableResult.RowsLoaded
			log.Printf("Successfully synced table %s: %d rows", table.PostgresTable, tableResult.RowsLoaded)
		}
	}
	
	result.EndTime = time.Now()
	
	log.Printf("Sync execution completed - Duration: %v, Total Rows: %d, Errors: %d", 
		result.EndTime.Sub(result.StartTime), result.TotalRows, result.TotalErrors)
	
	return result, nil
}

// TableConfig defines the mapping between PostgreSQL and Snowflake tables
type TableConfig struct {
	PostgresTable  string
	SnowflakeTable string
	OrderBy        string
}

// getTablesToSync returns the list of tables to synchronize
func (s *Synchronizer) getTablesToSync(requestedTables []string) ([]TableConfig, error) {
	// Define all available tables (matches the Python script)
	allTables := []TableConfig{
		{PostgresTable: "datacalls", SnowflakeTable: "ZTMF_DATACALLS", OrderBy: "datacallid"},
		{PostgresTable: "datacalls_fismasystems", SnowflakeTable: "ZTMF_DATACALLS_FISMASYSTEMS", OrderBy: "datacallid, fismasystemid"},
		{PostgresTable: "events", SnowflakeTable: "ZTMF_EVENTS", OrderBy: "createdat"},
		{PostgresTable: "fismasystems", SnowflakeTable: "ZTMF_FISMASYSTEMS", OrderBy: "fismasystemid"},
		{PostgresTable: "functionoptions", SnowflakeTable: "ZTMF_FUNCTIONOPTIONS", OrderBy: "functionoptionid"},
		{PostgresTable: "functions", SnowflakeTable: "ZTMF_FUNCTIONS", OrderBy: "functionid"},
		{PostgresTable: "massemails", SnowflakeTable: "ZTMF_MASSEMAILS", OrderBy: "massemailid"},
		{PostgresTable: "pillars", SnowflakeTable: "ZTMF_PILLARS", OrderBy: "pillarid"},
		{PostgresTable: "questions", SnowflakeTable: "ZTMF_QUESTIONS", OrderBy: "questionid"},
		{PostgresTable: "scores", SnowflakeTable: "ZTMF_SCORES", OrderBy: "scoreid"},
		{PostgresTable: "users", SnowflakeTable: "ZTMF_USERS", OrderBy: "userid"},
		{PostgresTable: "users_fismasystems", SnowflakeTable: "ZTMF_USERS_FISMASYSTEMS", OrderBy: "userid, fismasystemid"},
	}
	
	// If no specific tables requested, return all
	if len(requestedTables) == 0 {
		return allTables, nil
	}
	
	// Filter for requested tables
	var selectedTables []TableConfig
	for _, requestedTable := range requestedTables {
		found := false
		for _, table := range allTables {
			if table.PostgresTable == requestedTable || table.SnowflakeTable == requestedTable {
				selectedTables = append(selectedTables, table)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown table: %s", requestedTable)
		}
	}
	
	return selectedTables, nil
}

// syncTable synchronizes a single table
func (s *Synchronizer) syncTable(ctx context.Context, table TableConfig, fullRefresh bool) TableSyncResult {
	startTime := time.Now()
	
	result := TableSyncResult{
		PostgresTable:  table.PostgresTable,
		SnowflakeTable: table.SnowflakeTable,
	}
	
	log.Printf("Starting sync for table %s -> %s (FullRefresh: %t, DryRun: %t)", 
		table.PostgresTable, table.SnowflakeTable, fullRefresh, s.dryRun)
	
	if s.dryRun {
		// Real dry-run: extract real data + test Snowflake with transaction rollback
		log.Printf("DRY RUN: Testing real data extraction and Snowflake connectivity for %s", table.PostgresTable)
		
		// 1. Extract real data from PostgreSQL
		log.Printf("Extracting real data from PostgreSQL table: %s", table.PostgresTable)
		exportResult, err := s.pgClient.ExportTable(ctx, table.PostgresTable, table.OrderBy)
		if err != nil {
			result.Error = fmt.Errorf("dry-run failed: could not extract from %s: %w", table.PostgresTable, err)
			return result
		}
		
		result.RowsExtracted = exportResult.RowsExtracted
		log.Printf("DRY RUN: Successfully extracted %d real rows from %s", result.RowsExtracted, table.PostgresTable)
		
		// 2. Test Snowflake connectivity with transaction rollback
		log.Printf("DRY RUN: Testing Snowflake load with transaction rollback for %s", table.SnowflakeTable)
		loadResult, err := s.snowClient.LoadTableWithRollback(ctx, table.SnowflakeTable, exportResult.Data, fullRefresh)
		if err != nil {
			result.Error = fmt.Errorf("dry-run failed: Snowflake connectivity test failed for %s: %w", table.SnowflakeTable, err)
			return result
		}
		
		result.RowsLoaded = loadResult.RowsLoaded
		log.Printf("DRY RUN: Successfully tested Snowflake load (%d rows validated, transaction rolled back)", result.RowsLoaded)
		
		// 3. Verify row counts match
		if result.RowsExtracted != result.RowsLoaded {
			result.Error = fmt.Errorf("dry-run validation failed: extracted %d rows but Snowflake validated %d", 
				result.RowsExtracted, result.RowsLoaded)
			return result
		}
		
		log.Printf("DRY RUN: Validation complete - %d rows extracted = %d rows validated (rolled back)", 
			result.RowsExtracted, result.RowsLoaded)
	} else {
		// Real sync implementation
		
		// 1. Extract data from PostgreSQL
		log.Printf("Extracting data from PostgreSQL table: %s", table.PostgresTable)
		exportResult, err := s.pgClient.ExportTable(ctx, table.PostgresTable, table.OrderBy)
		if err != nil {
			result.Error = fmt.Errorf("failed to extract from %s: %w", table.PostgresTable, err)
			return result
		}
		
		result.RowsExtracted = exportResult.RowsExtracted
		
		// 2. Load data to Snowflake
		log.Printf("Loading %d rows to Snowflake table: %s", result.RowsExtracted, table.SnowflakeTable)
		loadResult, err := s.snowClient.LoadTable(ctx, table.SnowflakeTable, exportResult.Data, fullRefresh)
		if err != nil {
			result.Error = fmt.Errorf("failed to load to %s: %w", table.SnowflakeTable, err)
			return result
		}
		
		result.RowsLoaded = loadResult.RowsLoaded
		
		// 3. Verify row counts
		if result.RowsExtracted != result.RowsLoaded {
			result.Error = fmt.Errorf("row count mismatch: extracted %d, loaded %d", 
				result.RowsExtracted, result.RowsLoaded)
			return result
		}
		
		log.Printf("Verified row counts: %d extracted = %d loaded", result.RowsExtracted, result.RowsLoaded)
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// Summary returns a human-readable summary of sync results
func (r *SyncResult) Summary() string {
	duration := r.EndTime.Sub(r.StartTime)
	
	status := "COMPLETED"
	if r.DryRun {
		status = "DRY RUN COMPLETED"
	}
	if r.TotalErrors > 0 {
		status = "COMPLETED WITH ERRORS"
	}
	
	return fmt.Sprintf("%s: %d tables, %d total rows, %d errors, duration: %v", 
		status, len(r.TablesSync), r.TotalRows, r.TotalErrors, duration)
}
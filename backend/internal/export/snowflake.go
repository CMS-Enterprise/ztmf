package export

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/snowflakedb/gosnowflake"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

// SnowflakeClient handles connections to Snowflake for data loading
type SnowflakeClient struct {
	db  *sql.DB
	cfg *SnowflakeConfig
}

// SnowflakeConfig contains Snowflake connection parameters
type SnowflakeConfig struct {
	Account      string `json:"account"`
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`      // Optional: for password auth
	PrivateKey   string `json:"private_key,omitempty"`   // Optional: unencrypted PEM private key
	Warehouse    string `json:"warehouse"`
	Database     string `json:"database"`
	Schema       string `json:"schema"`
	Role         string `json:"role"`
	InsecureMode bool   `json:"insecure_mode,omitempty"` // Disable TLS verification (GovCloud endpoints)
}

// LoadResult contains the results of a data load operation
type LoadResult struct {
	Table       string
	RowsLoaded  int64
	Duration    time.Duration
	Error       error
}

// NewSnowflakeClient creates a new Snowflake client for data loading
func NewSnowflakeClient(ctx context.Context) (*SnowflakeClient, error) {
	// Build connection string from config or secrets
	snowflakeConfig, connString, err := buildSnowflakeConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to build Snowflake connection string: %w", err)
	}
	
	log.Printf("Connecting to Snowflake account: %s", snowflakeConfig.Account)
	
	// Open connection
	db, err := sql.Open("snowflake", connString)
	if err != nil {
		// Sanitize: connection string may contain credentials
		return nil, fmt.Errorf("failed to open Snowflake connection - check credentials and account configuration")
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		// Sanitize: driver errors may include DSN with credentials
		return nil, fmt.Errorf("failed to ping Snowflake - check credentials and network connectivity")
	}
	
	client := &SnowflakeClient{
		db:  db,
		cfg: snowflakeConfig,
	}
	
	// Set Snowflake session parameters
	if err := client.initializeSession(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize Snowflake session: %w", err)
	}
	
	log.Printf("Snowflake connection established successfully")
	
	return client, nil
}

// DB returns the underlying database connection for direct query access.
func (c *SnowflakeClient) DB() *sql.DB {
	return c.db
}

// Close closes the Snowflake connection
func (c *SnowflakeClient) Close() {
	if c.db != nil {
		log.Println("Closing Snowflake connection...")
		c.db.Close()
	}
}

// LoadTable loads data into a Snowflake table
func (c *SnowflakeClient) LoadTable(ctx context.Context, tableName string, data []map[string]interface{}, truncateFirst bool) (*LoadResult, error) {
	startTime := time.Now()

	result := &LoadResult{
		Table: tableName,
	}

	if err := ValidateTableIdentifier(tableName); err != nil {
		result.Error = fmt.Errorf("invalid table name: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}

	log.Printf("Loading data to Snowflake table: %s (rows: %d, truncate: %t)",
		tableName, len(data), truncateFirst)
	
	// Begin transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to begin transaction: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	defer tx.Rollback() // Will be no-op if transaction is committed
	
	// Truncate table if requested
	if truncateFirst {
		truncateSQL := fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", tableName)
		log.Printf("Truncating table: %s", truncateSQL)
		
		if _, err := tx.ExecContext(ctx, truncateSQL); err != nil {
			result.Error = fmt.Errorf("failed to truncate table %s: %w", tableName, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}
	}
	
	// If no data, just return success
	if len(data) == 0 {
		if err := tx.Commit(); err != nil {
			result.Error = fmt.Errorf("failed to commit empty transaction: %w", err)
		}
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	// Build INSERT statement from first row to get column names
	firstRow := data[0]
	columns := make([]string, 0, len(firstRow))
	placeholders := make([]string, 0, len(firstRow))
	
	// Get columns in consistent order
	for column := range firstRow {
		columns = append(columns, strings.ToUpper(column))
		placeholders = append(placeholders, "?")
	}
	
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	
	log.Printf("Preparing INSERT statement: %s", insertSQL)
	
	// Prepare statement
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		result.Error = fmt.Errorf("failed to prepare INSERT statement: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	defer stmt.Close()
	
	// Insert data in batches
	batchSize := 1000
	totalRows := int64(0)
	
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		
		batch := data[i:end]
		
		for _, row := range batch {
			// Extract values in same order as columns
			values := make([]interface{}, len(columns))
			for j, column := range columns {
				values[j] = row[strings.ToLower(column)]
			}
			
			if _, err := stmt.ExecContext(ctx, values...); err != nil {
				result.Error = fmt.Errorf("failed to insert row %d: %w", totalRows, err)
				result.Duration = time.Since(startTime)
				return result, result.Error
			}
			
			totalRows++
		}
		
		// Log progress for large loads
		if i > 0 && i%10000 == 0 {
			log.Printf("Loaded %d rows to %s...", i, tableName)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		result.Error = fmt.Errorf("failed to commit transaction: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	result.RowsLoaded = totalRows
	result.Duration = time.Since(startTime)
	
	log.Printf("Successfully loaded %d rows to %s (Duration: %v)", 
		totalRows, tableName, result.Duration)
	
	return result, nil
}

// MergeTable performs MERGE/upsert operation on a Snowflake table using parameterized bind variables.
// Data values are passed as bind parameters via SELECT ? UNION ALL SELECT ? in the USING clause,
// so no user data is ever concatenated into the SQL string.
func (c *SnowflakeClient) MergeTable(ctx context.Context, tableName string, data []map[string]interface{}, primaryKeys []string) (*LoadResult, error) {
	startTime := time.Now()

	result := &LoadResult{
		Table: tableName,
	}

	if err := ValidateTableIdentifier(tableName); err != nil {
		result.Error = fmt.Errorf("invalid table name: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}

	log.Printf("Merging data to Snowflake table: %s (rows: %d, keys: %v)", tableName, len(data), primaryKeys)

	// If no data, just return success
	if len(data) == 0 {
		result.Duration = time.Since(startTime)
		log.Printf("No data to merge for table %s", tableName)
		return result, nil
	}

	// Get column information from first row
	firstRow := data[0]
	columns := make([]string, 0, len(firstRow))
	for column := range firstRow {
		columns = append(columns, strings.ToUpper(column))
	}

	// Build the static parts of the MERGE statement (these don't change per batch)
	joinConditions := make([]string, len(primaryKeys))
	for i, key := range primaryKeys {
		keyUpper := strings.ToUpper(key)
		joinConditions[i] = fmt.Sprintf("target.%s = source.%s", keyUpper, keyUpper)
	}

	updateSets := make([]string, 0)
	for _, column := range columns {
		isPrimaryKey := false
		for _, key := range primaryKeys {
			if strings.ToUpper(key) == column {
				isPrimaryKey = true
				break
			}
		}
		if !isPrimaryKey {
			updateSets = append(updateSets, fmt.Sprintf("%s = source.%s", column, column))
		}
	}
	if len(updateSets) == 0 {
		updateSets = append(updateSets, fmt.Sprintf("%s = source.%s", columns[0], columns[0]))
	}

	insertColumns := strings.Join(columns, ", ")
	sourceInsertValues := make([]string, len(columns))
	for i, column := range columns {
		sourceInsertValues[i] = fmt.Sprintf("source.%s", column)
	}

	joinClause := strings.Join(joinConditions, " AND ")
	updateClause := strings.Join(updateSets, ", ")
	insertValuesClause := strings.Join(sourceInsertValues, ", ")

	// Build a single-row SELECT with ? placeholders for each column
	// e.g. "SELECT ? AS COL1, ? AS COL2"
	selectParts := make([]string, len(columns))
	for i, col := range columns {
		selectParts[i] = fmt.Sprintf("? AS %s", col)
	}
	firstRowSelect := "SELECT " + strings.Join(selectParts, ", ")

	// Subsequent rows use plain "SELECT ?, ?, ?" (no aliases needed in UNION ALL)
	plainPlaceholders := make([]string, len(columns))
	for i := range plainPlaceholders {
		plainPlaceholders[i] = "?"
	}
	subsequentRowSelect := "SELECT " + strings.Join(plainPlaceholders, ", ")

	// Process data in batches to keep SQL statement size reasonable
	batchSize := 100
	totalRows := int64(0)

	log.Printf("Processing %d rows in batches of %d", len(data), batchSize)

	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]

		// Build the USING subquery: SELECT ? AS COL1, ? AS COL2 UNION ALL SELECT ?, ? ...
		// Collect all bind parameter values in order
		bindArgs := make([]interface{}, 0, len(batch)*len(columns))
		rowSelects := make([]string, len(batch))

		for r, row := range batch {
			for _, column := range columns {
				bindArgs = append(bindArgs, row[strings.ToLower(column)])
			}
			if r == 0 {
				rowSelects[r] = firstRowSelect
			} else {
				rowSelects[r] = subsequentRowSelect
			}
		}

		usingSubquery := strings.Join(rowSelects, " UNION ALL ")

		mergeSQL := fmt.Sprintf(`MERGE INTO %s AS target
USING (%s) AS source
ON %s
WHEN MATCHED THEN
	UPDATE SET %s
WHEN NOT MATCHED THEN
	INSERT (%s) VALUES (%s)`,
			tableName,
			usingSubquery,
			joinClause,
			updateClause,
			insertColumns,
			insertValuesClause)

		log.Printf("Executing MERGE batch %d-%d (%d rows, %d bind params)",
			i+1, end, len(batch), len(bindArgs))

		if _, err := c.db.ExecContext(ctx, mergeSQL, bindArgs...); err != nil {
			result.Error = fmt.Errorf("failed to execute MERGE batch %d-%d: %w", i+1, end, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}

		totalRows += int64(len(batch))

		if end%500 == 0 || end == len(data) {
			log.Printf("Merged %d/%d rows...", end, len(data))
		}
	}

	result.RowsLoaded = totalRows
	result.Duration = time.Since(startTime)

	log.Printf("Successfully merged %d rows to %s (Duration: %v)",
		totalRows, tableName, result.Duration)

	return result, nil
}

// LoadTableWithRollback tests data loading to Snowflake with transaction rollback (for dry-run validation)
func (c *SnowflakeClient) LoadTableWithRollback(ctx context.Context, tableName string, data []map[string]interface{}, truncateFirst bool) (*LoadResult, error) {
	startTime := time.Now()

	result := &LoadResult{
		Table: tableName,
	}

	if err := ValidateTableIdentifier(tableName); err != nil {
		result.Error = fmt.Errorf("invalid table name: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}

	log.Printf("DRY RUN: Testing Snowflake load for table %s with transaction rollback (rows: %d, truncate: %t)",
		tableName, len(data), truncateFirst)
	
	// Begin transaction for rollback testing
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to begin rollback transaction: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	// Ensure rollback happens regardless of success/failure
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Warning: rollback failed: %v", rollbackErr)
		} else {
			log.Printf("DRY RUN: Transaction successfully rolled back for %s", tableName)
		}
	}()
	
	// Skip truncate testing in dry-run mode (may not have DELETE permissions)
	if truncateFirst {
		log.Printf("DRY RUN: Skipping truncate test (would truncate %s in real sync)", tableName)
	}
	
	// If no data, just test the table access
	if len(data) == 0 {
		// Test table access with SELECT
		testSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		var count int64
		if err := tx.QueryRowContext(ctx, testSQL).Scan(&count); err != nil {
			result.Error = fmt.Errorf("dry-run failed: could not access table %s: %w", tableName, err)
		}
		log.Printf("DRY RUN: Table access test successful for %s (current count: %d)", tableName, count)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	// Test INSERT with small batch (limit to 10 rows for dry-run performance)
	testData := data
	if len(data) > 10 {
		testData = data[:10]
		log.Printf("DRY RUN: Testing with first 10 rows out of %d total", len(data))
	}
	
	// Build INSERT statement from first row
	firstRow := testData[0]
	columns := make([]string, 0, len(firstRow))
	placeholders := make([]string, 0, len(firstRow))
	
	for column := range firstRow {
		columns = append(columns, strings.ToUpper(column))
		placeholders = append(placeholders, "?")
	}
	
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	
	log.Printf("DRY RUN: Testing INSERT statement: %s", insertSQL)
	
	// Prepare statement
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		result.Error = fmt.Errorf("dry-run failed: could not prepare INSERT for %s: %w", tableName, err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	defer stmt.Close()
	
	// Insert test data
	totalRows := int64(0)
	for _, row := range testData {
		// Extract values in same order as columns
		values := make([]interface{}, len(columns))
		for j, column := range columns {
			values[j] = row[strings.ToLower(column)]
		}
		
		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			result.Error = fmt.Errorf("dry-run failed: could not test insert row %d into %s: %w", totalRows, tableName, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}
		
		totalRows++
	}
	
	// Extrapolate row count (if we only tested 10 out of 1000, report 1000)
	result.RowsLoaded = int64(len(data))
	result.Duration = time.Since(startTime)
	
	log.Printf("DRY RUN: Successfully validated INSERT for %s (%d test rows, %d total would be loaded)", 
		tableName, totalRows, result.RowsLoaded)
	
	return result, nil
}

// DeleteExcludedRows removes rows from a Snowflake table whose primary key values
// are not present in the provided dataset. This handles the case where a source-side
// filter (e.g. sdl_sync_enabled toggled off) excludes rows that were previously synced.
// The MERGE operation only upserts; it cannot detect rows that disappeared from the
// source result set. This method fills that gap.
//
// data is the filtered export from PostgreSQL (only rows that SHOULD exist in Snowflake).
// Uses a NOT EXISTS subquery with bind parameters to avoid SQL injection.
func (c *SnowflakeClient) DeleteExcludedRows(ctx context.Context, tableName string, data []map[string]interface{}, primaryKeys []string) (int64, error) {
	if err := ValidateTableIdentifier(tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}

	if len(primaryKeys) == 0 {
		return 0, fmt.Errorf("primaryKeys must not be empty")
	}

	// If source returned zero rows, delete everything from the Snowflake table
	if len(data) == 0 {
		deleteSQL := fmt.Sprintf("DELETE FROM %s", tableName)
		log.Printf("WARNING: source filter returned 0 rows for %s — deleting ALL Snowflake rows. If this is unexpected, check SDL sync filters.", tableName)
		res, err := c.db.ExecContext(ctx, deleteSQL)
		if err != nil {
			return 0, fmt.Errorf("failed to delete all rows from %s: %w", tableName, err)
		}
		return res.RowsAffected()
	}

	keysUpper := make([]string, len(primaryKeys))
	keysLower := make([]string, len(primaryKeys))
	for i, k := range primaryKeys {
		keysUpper[i] = strings.ToUpper(k)
		keysLower[i] = strings.ToLower(k)
	}

	// Build join condition for NOT EXISTS: keep.K1 = target.K1 AND keep.K2 = target.K2
	joinConds := make([]string, len(keysUpper))
	for j, ku := range keysUpper {
		joinConds[j] = fmt.Sprintf("keep.%s = target.%s", ku, ku)
	}
	joinClause := strings.Join(joinConds, " AND ")

	// Build the USING subquery as SELECT ? AS K1, ? AS K2 UNION ALL SELECT ?, ? ...
	// Same pattern as MergeTable — first row uses aliases, subsequent rows use plain ?
	selectParts := make([]string, len(keysUpper))
	for i, ku := range keysUpper {
		selectParts[i] = fmt.Sprintf("? AS %s", ku)
	}
	firstRowSelect := "SELECT " + strings.Join(selectParts, ", ")

	plainPlaceholders := make([]string, len(keysUpper))
	for i := range plainPlaceholders {
		plainPlaceholders[i] = "?"
	}
	subsequentRowSelect := "SELECT " + strings.Join(plainPlaceholders, ", ")

	// Build the full USING subquery with ALL source rows in a single pass.
	// ZTMF tables are small (hundreds of rows), so a single statement is fine.
	allBindArgs := make([]interface{}, 0, len(data)*len(primaryKeys))
	allRowSelects := make([]string, len(data))

	for r, row := range data {
		for _, kl := range keysLower {
			allBindArgs = append(allBindArgs, row[kl])
		}
		if r == 0 {
			allRowSelects[r] = firstRowSelect
		} else {
			allRowSelects[r] = subsequentRowSelect
		}
	}

	usingSubquery := strings.Join(allRowSelects, " UNION ALL ")

	deleteSQL := fmt.Sprintf(`DELETE FROM %s AS target WHERE NOT EXISTS (
		SELECT 1 FROM (%s) AS keep WHERE %s
	)`, tableName, usingSubquery, joinClause)

	log.Printf("Deleting excluded rows from %s (%d source rows to keep)", tableName, len(data))

	res, err := c.db.ExecContext(ctx, deleteSQL, allBindArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete excluded rows from %s: %w", tableName, err)
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		log.Printf("Warning: could not determine rows deleted from %s: %v", tableName, err)
	}

	if deleted > 0 {
		log.Printf("Deleted %d stale rows from %s", deleted, tableName)
	} else {
		log.Printf("No stale rows to delete from %s", tableName)
	}

	return deleted, nil
}

// GetTableRowCount returns the number of rows in a Snowflake table
func (c *SnowflakeClient) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if err := ValidateTableIdentifier(tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	var count int64
	err := c.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count for %s: %w", tableName, err)
	}
	
	return count, nil
}

// TestConnection tests the Snowflake connection
func (c *SnowflakeClient) TestConnection(ctx context.Context) error {
	query := "SELECT 1 as test"
	
	var result int
	err := c.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	
	if result != 1 {
		return fmt.Errorf("unexpected test result: got %d, expected 1", result)
	}
	
	log.Printf("Snowflake connection test successful")
	return nil
}

// initializeSession sets up the Snowflake session with proper context.
// Role is set first because it determines access to warehouse/database/schema.
func (c *SnowflakeClient) initializeSession(ctx context.Context) error {
	// Validate and sanitize inputs to prevent SQL injection
	if err := c.validateSessionIdentifiers(); err != nil {
		return fmt.Errorf("invalid session parameters: %w", err)
	}

	// Set role FIRST — it determines permissions for subsequent USE commands
	if c.cfg.Role != "" {
		if err := c.executeSessionCommand(ctx, "USE ROLE", c.cfg.Role); err != nil {
			return err
		}
	}

	if c.cfg.Warehouse != "" {
		if err := c.executeSessionCommand(ctx, "USE WAREHOUSE", c.cfg.Warehouse); err != nil {
			return err
		}
	}

	if c.cfg.Database != "" {
		if err := c.executeSessionCommand(ctx, "USE DATABASE", c.cfg.Database); err != nil {
			return err
		}
	}

	if c.cfg.Schema != "" {
		if err := c.executeSessionCommand(ctx, "USE SCHEMA", c.cfg.Schema); err != nil {
			return err
		}
	}

	return nil
}

// validateSessionIdentifiers ensures identifiers are safe for SQL execution
func (c *SnowflakeClient) validateSessionIdentifiers() error {
	identifiers := map[string]string{
		"warehouse": c.cfg.Warehouse,
		"database":  c.cfg.Database,
		"schema":    c.cfg.Schema,
		"role":      c.cfg.Role,
	}
	
	for name, value := range identifiers {
		if value != "" && !isValidSnowflakeIdentifier(value) {
			return fmt.Errorf("invalid %s identifier: %s", name, value)
		}
	}
	
	return nil
}

// executeSessionCommand safely executes a session command with validated identifier
func (c *SnowflakeClient) executeSessionCommand(ctx context.Context, command, identifier string) error {
	// Use prepared statements to prevent SQL injection
	var sql string
	var args []interface{}
	
	// Use direct SQL commands like the working Python script (not IDENTIFIER function)
	switch command {
	case "USE WAREHOUSE":
		sql = "USE WAREHOUSE " + identifier
	case "USE DATABASE":
		sql = "USE DATABASE " + identifier  
	case "USE SCHEMA":
		sql = "USE SCHEMA " + identifier
	case "USE ROLE":
		sql = "USE ROLE " + identifier
	default:
		return fmt.Errorf("unsupported session command: %s", command)
	}
	
	// No args needed for direct identifier usage
	args = nil
	
	if _, err := c.db.ExecContext(ctx, sql, args...); err != nil {
		return fmt.Errorf("failed to execute session command '%s': %w", sql, err)
	}
	
	return nil
}

// ValidateTableIdentifier checks that a table name (possibly qualified like SCHEMA.TABLE)
// contains only safe identifier characters. Returns an error if the name is invalid.
func ValidateTableIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	// Split on dots for qualified names like DATABASE.SCHEMA.TABLE
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if !isValidSnowflakeIdentifier(part) {
			return fmt.Errorf("invalid identifier component %q in table name %q", part, name)
		}
	}
	return nil
}

// isValidSnowflakeIdentifier checks if an identifier is safe (alphanumeric, underscore, dash)
func isValidSnowflakeIdentifier(identifier string) bool {
	if len(identifier) == 0 || len(identifier) > 255 {
		return false
	}
	
	// Allow alphanumeric, underscore, dash (standard Snowflake identifiers)
	for _, char := range identifier {
		if !((char >= 'A' && char <= 'Z') || 
			 (char >= 'a' && char <= 'z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_' || char == '-') {
			return false
		}
	}
	
	return true
}


// buildSnowflakeConnectionString creates a Snowflake connection string from config or secrets
func buildSnowflakeConnectionString() (*SnowflakeConfig, string, error) {
	cfg := config.GetInstance()
	
	// Try to load from secrets first
	var snowflakeConfig *SnowflakeConfig
	
	// Look for Snowflake secret in environment
	secretID := ""
	if cfg.Env == "prod" {
		secretID = "ztmf_snowflake_prod"
	} else {
		secretID = "ztmf_snowflake_dev"
	}
	
	log.Printf("Loading Snowflake credentials from secret: %s", secretID)
	
	snowflakeSecret, err := secrets.NewSecret(secretID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load Snowflake secret: %w", err)
	}
	
	snowflakeConfig = &SnowflakeConfig{}
	if err := snowflakeSecret.Unmarshal(snowflakeConfig); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal Snowflake secret: %w", err)
	}
	
	// Validate required fields
	if snowflakeConfig.Account == "" || snowflakeConfig.Username == "" {
		return nil, "", fmt.Errorf("missing required Snowflake credentials (account, username)")
	}
	
	// Ensure at least one authentication method is provided
	hasPassword := snowflakeConfig.Password != ""
	hasRSAKey := snowflakeConfig.PrivateKey != ""
	
	if !hasPassword && !hasRSAKey {
		return nil, "", fmt.Errorf("authentication required: provide either password OR private_key")
	}
	
	// Set defaults if not provided (skip warehouse - let service account use default)
	if snowflakeConfig.Database == "" {
		snowflakeConfig.Database = "BUS_ZEROTRUST"
	}
	if snowflakeConfig.Schema == "" {
		snowflakeConfig.Schema = "PRIVATE"
	}
	if snowflakeConfig.Role == "" {
		snowflakeConfig.Role = "ZTMF_LOADER"
	}
	
	// Build connection configuration based on authentication type  
	if snowflakeConfig.PrivateKey != "" {
		// RSA key authentication (using unencrypted PEM)
		log.Printf("Using RSA key authentication for Snowflake")
		
		// Parse PEM private key
		block, _ := pem.Decode([]byte(snowflakeConfig.PrivateKey))
		if block == nil {
			return nil, "", fmt.Errorf("failed to parse PEM block from private key")
		}
		
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, "", fmt.Errorf("private key is not an RSA key")
		}
		
		// Use gosnowflake config struct for RSA authentication
		// Start with minimal connection, then set context
		cfg := &gosnowflake.Config{
			Account:           snowflakeConfig.Account,
			User:              snowflakeConfig.Username,
			Password:          "", // Required by driver even for RSA auth
			PrivateKey:        rsaPrivateKey,
			Authenticator:     gosnowflake.AuthTypeJwt,
			InsecureMode:      snowflakeConfig.InsecureMode,
		}
		
		connString, err := gosnowflake.DSN(cfg)
		if err != nil {
			return nil, "", fmt.Errorf("failed to build RSA connection string: %w", err)
		}
		
		return snowflakeConfig, connString, nil
	} else {
		// Password authentication (fallback)
		log.Printf("Using password authentication for Snowflake")
		connString := fmt.Sprintf("%s:%s@%s/%s/%s?warehouse=%s&role=%s",
			snowflakeConfig.Username,
			snowflakeConfig.Password,
			snowflakeConfig.Account,
			snowflakeConfig.Database,
			snowflakeConfig.Schema,
			snowflakeConfig.Warehouse,
			snowflakeConfig.Role)

		if snowflakeConfig.InsecureMode {
			connString += "&insecureMode=true"
		}

		return snowflakeConfig, connString, nil
	}
}


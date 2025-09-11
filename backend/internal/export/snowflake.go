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
	Account    string `json:"account"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`    // Optional: for password auth
	PrivateKey string `json:"private_key,omitempty"` // Optional: unencrypted PEM private key
	Warehouse  string `json:"warehouse"`
	Database   string `json:"database"`
	Schema     string `json:"schema"`
	Role       string `json:"role"`
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
		return nil, fmt.Errorf("failed to open Snowflake connection: %w", err)
	}
	
	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping Snowflake: %w", err)
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

// LoadTableWithRollback tests data loading to Snowflake with transaction rollback (for dry-run validation)
func (c *SnowflakeClient) LoadTableWithRollback(ctx context.Context, tableName string, data []map[string]interface{}, truncateFirst bool) (*LoadResult, error) {
	startTime := time.Now()
	
	result := &LoadResult{
		Table: tableName,
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
	
	// Test truncate if requested
	if truncateFirst {
		truncateSQL := fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", tableName)
		log.Printf("DRY RUN: Testing truncate: %s", truncateSQL)
		
		if _, err := tx.ExecContext(ctx, truncateSQL); err != nil {
			result.Error = fmt.Errorf("dry-run failed: could not test truncate table %s: %w", tableName, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}
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

// GetTableRowCount returns the number of rows in a Snowflake table
func (c *SnowflakeClient) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
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

// initializeSession sets up the Snowflake session with proper context
func (c *SnowflakeClient) initializeSession(ctx context.Context) error {
	sessionCommands := []string{
		fmt.Sprintf("USE WAREHOUSE %s", c.cfg.Warehouse),
		fmt.Sprintf("USE DATABASE %s", c.cfg.Database),
		fmt.Sprintf("USE SCHEMA %s", c.cfg.Schema),
		fmt.Sprintf("USE ROLE %s", c.cfg.Role),
	}
	
	for _, cmd := range sessionCommands {
		log.Printf("Executing Snowflake session command: %s", cmd)
		if _, err := c.db.ExecContext(ctx, cmd); err != nil {
			return fmt.Errorf("failed to execute session command '%s': %w", cmd, err)
		}
	}
	
	return nil
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
	
	// Set defaults if not provided
	if snowflakeConfig.Warehouse == "" {
		snowflakeConfig.Warehouse = "TEAM_ZERO_TRUST_WH"
	}
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
		cfg := &gosnowflake.Config{
			Account:    snowflakeConfig.Account,
			User:       snowflakeConfig.Username,
			PrivateKey: rsaPrivateKey,
			Database:   snowflakeConfig.Database,
			Schema:     snowflakeConfig.Schema,
			Warehouse:  snowflakeConfig.Warehouse,
			Role:       snowflakeConfig.Role,
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
		
		return snowflakeConfig, connString, nil
	}
}


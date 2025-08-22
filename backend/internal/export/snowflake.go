package export

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/snowflakedb/gosnowflake"

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
	Account   string `json:"account"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Warehouse string `json:"warehouse"`
	Database  string `json:"database"`
	Schema    string `json:"schema"`
	Role      string `json:"role"`
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
	if snowflakeConfig.Account == "" || snowflakeConfig.Username == "" || snowflakeConfig.Password == "" {
		return nil, "", fmt.Errorf("missing required Snowflake credentials (account, username, password)")
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
	
	// Build connection string
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
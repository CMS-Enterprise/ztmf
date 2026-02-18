package export

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

// PostgresClient handles connections to PostgreSQL for data extraction
type PostgresClient struct {
	pool *pgxpool.Pool
}

// ExportResult contains the results of a data export operation
type ExportResult struct {
	Table         string
	RowsExtracted int64
	Duration      time.Duration
	Error         error
	Data          []map[string]interface{}
}

// NewPostgresClient creates a new PostgreSQL client for data export
func NewPostgresClient(ctx context.Context) (*PostgresClient, error) {
	// Build connection string from config or secrets
	connString, err := buildConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to build connection string: %w", err)
	}
	
	log.Printf("Connecting to PostgreSQL for data export...")
	
	// Create connection pool
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		// Sanitize error to prevent credential exposure in logs
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool - check database credentials and connectivity")
	}
	
	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	log.Printf("PostgreSQL connection established successfully")
	
	return &PostgresClient{
		pool: pool,
	}, nil
}

// Pool returns the underlying connection pool for direct query access.
func (c *PostgresClient) Pool() *pgxpool.Pool {
	return c.pool
}

// Close closes the PostgreSQL connection pool
func (c *PostgresClient) Close() {
	if c.pool != nil {
		log.Println("Closing PostgreSQL connection pool...")
		c.pool.Close()
	}
}

// ExportTable extracts all data from a PostgreSQL table
func (c *PostgresClient) ExportTable(ctx context.Context, tableName string, orderBy string) (*ExportResult, error) {
	return c.ExportTableWhere(ctx, tableName, orderBy, "")
}

// ExportTableWhere extracts data from a PostgreSQL table with an optional WHERE clause.
// The whereClause parameter should be a valid SQL condition without the WHERE keyword
// (e.g. "sdl_sync_enabled = true"). If empty, all rows are returned.
func (c *PostgresClient) ExportTableWhere(ctx context.Context, tableName string, orderBy string, whereClause string) (*ExportResult, error) {
	startTime := time.Now()

	result := &ExportResult{
		Table: tableName,
	}

	if err := ValidateTableIdentifier(tableName); err != nil {
		result.Error = fmt.Errorf("invalid table name: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	if orderBy != "" {
		// orderBy can be comma-separated column list like "datacallid, fismasystemid"
		for _, col := range strings.Split(orderBy, ",") {
			col = strings.TrimSpace(col)
			if col != "" {
				if err := ValidateTableIdentifier(col); err != nil {
					result.Error = fmt.Errorf("invalid orderBy column: %w", err)
					result.Duration = time.Since(startTime)
					return result, result.Error
				}
			}
		}
	}

	log.Printf("Exporting data from table: %s", tableName)

	// Build query with optional WHERE and ORDER BY
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	if orderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", orderBy)
	}

	// Execute query
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		result.Error = fmt.Errorf("failed to execute query: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = fd.Name
	}


	// Extract all rows
	var data []map[string]interface{}
	rowCount := int64(0)

	for rows.Next() {
		// Scan row into interface{} values
		values, err := rows.Values()
		if err != nil {
			result.Error = fmt.Errorf("failed to scan row %d: %w", rowCount, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}

		// Build row map
		rowData := make(map[string]interface{})
		for i, columnName := range columnNames {
			rowData[columnName] = values[i]
		}

		data = append(data, rowData)
		rowCount++

		// Log progress for large tables
		if rowCount%10000 == 0 {
			log.Printf("Extracted %d rows from %s...", rowCount, tableName)
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		result.Error = fmt.Errorf("error during row iteration: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}

	result.RowsExtracted = rowCount
	result.Data = data
	result.Duration = time.Since(startTime)

	log.Printf("Successfully extracted %d rows from %s (Duration: %v)",
		rowCount, tableName, result.Duration)

	return result, nil
}

// GetTableRowCount returns the number of rows in a table
func (c *PostgresClient) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if err := ValidateTableIdentifier(tableName); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	var count int64
	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count for %s: %w", tableName, err)
	}
	
	return count, nil
}

// buildConnectionString creates a PostgreSQL connection string from config or secrets
func buildConnectionString() (string, error) {
	cfg := config.GetInstance()
	// If we have all direct config values, use them
	if cfg.Db.Host != "" && cfg.Db.User != "" && cfg.Db.Pass != "" {
		// URL encode credentials to handle special characters safely
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=prefer",
			url.QueryEscape(cfg.Db.User), 
			url.QueryEscape(cfg.Db.Pass), 
			cfg.Db.Host, cfg.Db.Port, cfg.Db.Name), nil
	}
	
	// If we have a secret ID, use that
	if cfg.Db.SecretId != "" {
		log.Printf("Loading database credentials from secret: %s", cfg.Db.SecretId)
		
		dbSecret, err := secrets.NewSecret(cfg.Db.SecretId)
		if err != nil {
			return "", fmt.Errorf("failed to load database secret: %w", err)
		}
		
		// Use same structure as existing API (only username/password in secret)
		type dbCreds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		
		var creds dbCreds
		if err := dbSecret.Unmarshal(&creds); err != nil {
			return "", fmt.Errorf("failed to unmarshal database secret: %w", err)
		}
		
		// Build connection string using config for host/port/database (like API does)
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=prefer",
			url.QueryEscape(creds.Username), 
			url.QueryEscape(creds.Password), 
			cfg.Db.Host, cfg.Db.Port, cfg.Db.Name), nil
	}
	
	return "", fmt.Errorf("insufficient database configuration - need either direct config or secret ID")
}
package export

import (
	"context"
	"fmt"
	"log"
	"net/url"
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

// Close closes the PostgreSQL connection pool
func (c *PostgresClient) Close() {
	if c.pool != nil {
		log.Println("Closing PostgreSQL connection pool...")
		c.pool.Close()
	}
}

// ExportTable extracts all data from a PostgreSQL table
func (c *PostgresClient) ExportTable(ctx context.Context, tableName string, orderBy string) (*ExportResult, error) {
	startTime := time.Now()
	
	result := &ExportResult{
		Table: tableName,
	}
	
	log.Printf("Exporting data from table: %s", tableName)
	
	// Build query with optional ORDER BY
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	if orderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", orderBy)
	}
	
	log.Printf("Executing query: %s", query)
	
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
	
	log.Printf("Found %d columns: %v", len(columnNames), columnNames)
	
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
		
		// Define structure for database secret
		type dbCreds struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		
		var creds dbCreds
		if err := dbSecret.Unmarshal(&creds); err != nil {
			return "", fmt.Errorf("failed to unmarshal database secret: %w", err)
		}
		
		// URL encode credentials to handle special characters safely
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=prefer",
			url.QueryEscape(creds.Username), 
			url.QueryEscape(creds.Password), 
			creds.Host, creds.Port, creds.Database), nil
	}
	
	return "", fmt.Errorf("insufficient database configuration - need either direct config or secret ID")
}
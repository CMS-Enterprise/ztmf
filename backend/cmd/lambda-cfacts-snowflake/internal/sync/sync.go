package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/cfacts"
	"github.com/CMS-Enterprise/ztmf/backend/internal/export"
)

// Synchronizer handles CFACTS data sync from Snowflake to PostgreSQL.
type Synchronizer struct {
	dryRun        bool
	snowflakeView string
	pgClient      *export.PostgresClient
	snowClient    *export.SnowflakeClient
}

// NewSynchronizer creates a new CFACTS Snowflake synchronizer.
func NewSynchronizer(ctx context.Context, dryRun bool) (*Synchronizer, error) {
	log.Printf("Initializing CFACTS Snowflake synchronizer - DryRun: %t", dryRun)

	snowflakeView := os.Getenv("CFACTS_SNOWFLAKE_VIEW")
	if snowflakeView == "" {
		return nil, fmt.Errorf("CFACTS_SNOWFLAKE_VIEW environment variable is required")
	}
	if err := export.ValidateTableIdentifier(snowflakeView); err != nil {
		return nil, fmt.Errorf("invalid CFACTS_SNOWFLAKE_VIEW: %w", err)
	}

	pgClient, err := export.NewPostgresClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL client: %w", err)
	}

	snowClient, err := export.NewSnowflakeClient(ctx)
	if err != nil {
		pgClient.Close()
		return nil, fmt.Errorf("failed to initialize Snowflake client: %w", err)
	}

	return &Synchronizer{
		dryRun:        dryRun,
		snowflakeView: snowflakeView,
		pgClient:      pgClient,
		snowClient:    snowClient,
	}, nil
}

// Close cleans up database connections.
func (s *Synchronizer) Close() {
	log.Println("Closing CFACTS Snowflake synchronizer connections...")
	if s.pgClient != nil {
		s.pgClient.Close()
	}
	if s.snowClient != nil {
		s.snowClient.Close()
	}
}

// ExecuteSync queries the Snowflake view and syncs results to PostgreSQL.
func (s *Synchronizer) ExecuteSync(ctx context.Context) (*cfacts.SyncResult, error) {
	log.Printf("Querying Snowflake view: %s", s.snowflakeView)

	systems, err := s.querySnowflake(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query Snowflake: %w", err)
	}

	log.Printf("Retrieved %d systems from Snowflake", len(systems))

	result, err := cfacts.SyncToPostgres(ctx, s.pgClient.Pool(), systems, s.dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to sync to PostgreSQL: %w", err)
	}

	return result, nil
}

// snowflakeColumns returns the explicit list of Snowflake column names we need.
// Only these columns are queried — the view may have more, but we ignore them.
// To sync a new column: add it to SnowflakeColumnMap, cfactsColumns, and CfactsSystem.
func snowflakeColumns() []string {
	cols := make([]string, 0, len(cfacts.SnowflakeColumnMap))
	for sfCol := range cfacts.SnowflakeColumnMap {
		cols = append(cols, sfCol)
	}
	return cols
}

// querySnowflake fetches only the columns we need from the CFACTS Snowflake view.
func (s *Synchronizer) querySnowflake(ctx context.Context) ([]cfacts.CfactsSystem, error) {
	cols := snowflakeColumns()
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), s.snowflakeView)

	log.Printf("Executing query with %d explicit columns", len(cols))

	rows, err := s.snowClient.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Build column index map from the columns we requested
	colIdx := make(map[string]int, len(cols))
	for i, col := range cols {
		colIdx[col] = i
	}

	var systems []cfacts.CfactsSystem
	rowNum := 0

	for rows.Next() {
		rowNum++

		// Scan targets match our explicit column list exactly
		scanValues := make([]any, len(cols))
		for i := range scanValues {
			scanValues[i] = new(sql.NullString)
		}

		if err := rows.Scan(scanValues...); err != nil {
			return nil, fmt.Errorf("failed to scan row %d: %w", rowNum, err)
		}

		sys, err := scanToSystem(scanValues, colIdx, rowNum)
		if err != nil {
			return nil, err
		}
		systems = append(systems, sys)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return systems, nil
}

// scanToSystem converts scanned SQL values into a CfactsSystem struct.
func scanToSystem(values []any, colIdx map[string]int, rowNum int) (cfacts.CfactsSystem, error) {
	getString := func(col string) string {
		idx, ok := colIdx[col]
		if !ok {
			return ""
		}
		ns := values[idx].(*sql.NullString)
		if !ns.Valid {
			return ""
		}
		return ns.String
	}

	uuid := getString("FISMA_UUID")
	if uuid == "" {
		return cfacts.CfactsSystem{}, fmt.Errorf("row %d: FISMA_UUID is empty", rowNum)
	}

	acronym := getString("FISMA_ACRONYM")
	if acronym == "" {
		return cfacts.CfactsSystem{}, fmt.Errorf("row %d: FISMA_ACRONYM is empty", rowNum)
	}

	sys := cfacts.CfactsSystem{
		FismaUUID:                uuid,
		FismaAcronym:             acronym,
		AuthorizationPackageName: optStr(getString("AUTHORIZATION_PACKAGE_NAME")),
		PrimaryISSOName:          optStr(getString("PRIMARY_ISSO_NAME")),
		PrimaryISSOEmail:         optStr(getString("PRIMARY_ISSO_EMAIL")),
		IsActive:                 optBoolFromStr(getString("IS_ACTIVE")),
		IsRetired:                optBoolFromStr(getString("IS_RETIRED")),
		IsDecommissioned:         optBoolFromStr(getString("IS_DECOMMISSIONED")),
		LifecyclePhase:           optStr(getString("LIFECYCLE_PHASE")),
		ComponentAcronym:         optStr(getString("COMPONENT_ACRONYM")),
		DivisionName:             optStr(getString("DIVISION_NAME")),
		GroupName:                optStr(getString("GROUP_NAME")),
	}

	var err error
	sys.ATOExpirationDate, err = optTimeFromStr(getString("ATO_EXPIRATION_DATE"))
	if err != nil {
		return cfacts.CfactsSystem{}, fmt.Errorf("row %d: bad ATO_EXPIRATION_DATE: %w", rowNum, err)
	}
	sys.DecommissionDate, err = optTimeFromStr(getString("DECOMMISSION_DATE"))
	if err != nil {
		return cfacts.CfactsSystem{}, fmt.Errorf("row %d: bad DECOMMISSION_DATE: %w", rowNum, err)
	}
	sys.LastModifiedDate, err = optTimeFromStr(getString("LAST_MODIFIED_DATE"))
	if err != nil {
		return cfacts.CfactsSystem{}, fmt.Errorf("row %d: bad LAST_MODIFIED_DATE: %w", rowNum, err)
	}

	return sys, nil
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optBoolFromStr(s string) *bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	b := s == "true" || s == "1"
	return &b
}

var timeFormats = []string{
	"2006-01-02 15:04:05.000000000",
	"2006-01-02 15:04:05.000000",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02",
}

func optTimeFromStr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("cannot parse timestamp %q", s)
}

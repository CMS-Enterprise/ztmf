package model

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// DataCenterEnvironment mirrors a row from the public.datacenterenvironments
// reference table. It maps the raw value stored on a system to a reporting
// category and the functions.datacenterenvironment set used to score it, so the
// scoring vocabulary lives in data rather than code (ztmf#392).
type DataCenterEnvironment struct {
	DataCenterEnvironment string `json:"datacenterenvironment" db:"datacenterenvironment"`
	Category              string `json:"category" db:"category"`
	// ScoringKey is the functions.datacenterenvironment set this environment is
	// scored against. NULL means the environment is not scored (e.g. the legacy
	// DECOMMISSIONED marker), so it is a pointer to distinguish that from "".
	ScoringKey *string `json:"scoring_key" db:"scoring_key"`
	Selectable bool    `json:"selectable" db:"selectable"`
	Ordr       int     `json:"ordr" db:"ordr"`
}

var dataCenterEnvironmentColumns = []string{"datacenterenvironment", "category", "scoring_key", "selectable", "ordr"}

// FindDataCenterEnvironmentsInput holds optional filters for listing environments.
type FindDataCenterEnvironmentsInput struct {
	// SelectableOnly restricts the result to values offered in the system
	// dropdown (selectable = TRUE). Used by the frontend to build the picker.
	SelectableOnly *bool `schema:"selectable_only"`
}

// FindDataCenterEnvironments returns the datacenterenvironments reference rows.
// The frontend calls it with SelectableOnly to build the system-environment
// dropdown so that list is backend-driven instead of hardcoded in the UI.
func FindDataCenterEnvironments(ctx context.Context, input FindDataCenterEnvironmentsInput) ([]*DataCenterEnvironment, error) {
	sqlb := stmntBuilder.
		Select(dataCenterEnvironmentColumns...).
		From("public.datacenterenvironments").
		OrderBy("ordr, category")

	if input.SelectableOnly != nil && *input.SelectableOnly {
		sqlb = sqlb.Where("selectable = ?", true)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCenterEnvironment])
}

// dataCenterEnvironmentExists reports whether raw is a known system environment
// value (any row in the mapping table). Used to validate fismasystems writes
// against the reference data instead of a compiled-in vocabulary.
func dataCenterEnvironmentExists(ctx context.Context, raw string) (bool, error) {
	return existsIn(ctx, "datacenterenvironment", raw)
}

// isValidScoringKey reports whether key is used as a scoring target by at least
// one mapping row, i.e. a legal value for functions.datacenterenvironment. Used
// to validate functions catalog writes.
func isValidScoringKey(ctx context.Context, key string) (bool, error) {
	return existsIn(ctx, "scoring_key", key)
}

// existsIn runs a single-column EXISTS check against the mapping table. col is a
// trusted internal constant (never user input), val is bound as a parameter.
func existsIn(ctx context.Context, col, val string) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, trapError(err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM public.datacenterenvironments WHERE "+col+" = $1)",
		val).Scan(&exists)
	if err != nil {
		return false, trapError(err)
	}
	return exists, nil
}

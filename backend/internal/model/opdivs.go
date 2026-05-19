package model

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// OpDiv mirrors a row from the public.opdivs reference table.
type OpDiv struct {
	OpDivID  int32  `json:"opdiv_id" db:"opdiv_id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsParent bool   `json:"is_parent" db:"is_parent"`
	Active   bool   `json:"active"`
}

// FindOpDivsInput holds optional filters for listing OpDivs.
type FindOpDivsInput struct {
	// ActiveOnly returns only opdivs.active = TRUE rows. Default true.
	ActiveOnly *bool `schema:"active_only"`
}

// FindOpDivs returns the OpDiv list. Used by the admin panel for dropdowns
// and by the onboarding workbook importer to validate opdiv codes.
func FindOpDivs(ctx context.Context, input FindOpDivsInput) ([]*OpDiv, error) {
	sqlb := stmntBuilder.
		Select("opdiv_id", "code", "name", "is_parent", "active").
		From("public.opdivs").
		OrderBy("(NOT is_parent), code")

	activeOnly := true
	if input.ActiveOnly != nil {
		activeOnly = *input.ActiveOnly
	}
	if activeOnly {
		sqlb = sqlb.Where("active = ?", true)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[OpDiv])
}

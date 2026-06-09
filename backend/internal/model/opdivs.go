package model

import (
	"context"
	"errors"
	"strings"

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

func (o *OpDiv) validate() error {
	// Canonicalize before checking AND storing so the length bound can't be
	// bypassed with padding and the active-code uniqueness index (LOWER(code),
	// not trimmed) cannot be defeated by leading/trailing whitespace.
	o.Code = strings.TrimSpace(o.Code)
	o.Name = strings.TrimSpace(o.Name)

	inputErr := InvalidInputError{data: map[string]any{}}
	if o.Code == "" || len(o.Code) > 16 {
		inputErr.data["code"] = "1-16 characters required"
	}
	if o.Name == "" || len(o.Name) > 128 {
		inputErr.data["name"] = "1-128 characters required"
	}

	if len(inputErr.data) > 0 {
		return &inputErr
	}
	return nil
}

// Save inserts a new OpDiv or updates an existing one (OpDivID > 0). A new
// OpDiv is always created active; deactivation is done by updating an existing
// row with active=false. The partial unique index on LOWER(code) WHERE active
// means inserting a second active row with an existing code returns ErrNotUnique.
func (o *OpDiv) Save(ctx context.Context) (*OpDiv, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}

	var sqlb SqlBuilder
	if o.OpDivID == 0 {
		sqlb = stmntBuilder.
			Insert("public.opdivs").
			Columns("code", "name", "is_parent", "active").
			Values(o.Code, o.Name, o.IsParent, true).
			Suffix("RETURNING opdiv_id, code, name, is_parent, active")
	} else {
		sqlb = stmntBuilder.
			Update("public.opdivs").
			Set("code", o.Code).
			Set("name", o.Name).
			Set("is_parent", o.IsParent).
			Set("active", o.Active).
			Where("opdiv_id=?", o.OpDivID).
			Suffix("RETURNING opdiv_id, code, name, is_parent, active")
	}

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByName[OpDiv])
	if err != nil {
		// The only unique constraint on opdivs is the active-code index, so map
		// a uniqueness violation to a clear field-level message the frontend can
		// show inline rather than leaking the raw constraint detail.
		if errors.Is(err, ErrNotUnique) {
			return nil, &InvalidInputError{data: map[string]any{"code": "an OpDiv with this code already exists"}}
		}
		return nil, err
	}
	return saved, nil
}

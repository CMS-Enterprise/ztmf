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
	// IsParent and Active are pointers so an update can distinguish "not supplied"
	// (nil, leave the column untouched) from an explicit false. With a non-pointer
	// bool, any update that omits the field defaults to false: omitting is_parent
	// would demote the HHS parent OpDiv (which drives HHS-wide authorization), and
	// omitting active would silently deactivate an active OpDiv.
	IsParent *bool `json:"is_parent" db:"is_parent"`
	Active   *bool `json:"active"`
	// InsightsEnabled gates whether system_enrichment (ZTMF Insights) is served
	// for systems in this OpDiv. Pointer for the same reason as IsParent/Active:
	// an update that omits it must leave the column untouched, not reset it to
	// false (which would silently disable enrichment for an enabled OpDiv).
	InsightsEnabled *bool `json:"insights_enabled" db:"insights_enabled"`
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
		Select("opdiv_id", "code", "name", "is_parent", "active", "insights_enabled").
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
		// Create: a new OpDiv is always active; is_parent and insights_enabled
		// default to false when the optional fields are omitted.
		isParent := o.IsParent != nil && *o.IsParent
		insightsEnabled := o.InsightsEnabled != nil && *o.InsightsEnabled
		sqlb = stmntBuilder.
			Insert("public.opdivs").
			Columns("code", "name", "is_parent", "active", "insights_enabled").
			Values(o.Code, o.Name, isParent, true, insightsEnabled).
			Suffix("RETURNING opdiv_id, code, name, is_parent, active, insights_enabled")
	} else {
		ub := stmntBuilder.
			Update("public.opdivs").
			Set("code", o.Code).
			Set("name", o.Name)
		// Only touch is_parent / active / insights_enabled when the caller
		// explicitly supplied them; omitting a field (nil) must leave the current
		// value intact.
		if o.IsParent != nil {
			ub = ub.Set("is_parent", *o.IsParent)
		}
		if o.Active != nil {
			ub = ub.Set("active", *o.Active)
		}
		if o.InsightsEnabled != nil {
			ub = ub.Set("insights_enabled", *o.InsightsEnabled)
		}
		sqlb = ub.
			Where("opdiv_id=?", o.OpDivID).
			Suffix("RETURNING opdiv_id, code, name, is_parent, active, insights_enabled")
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

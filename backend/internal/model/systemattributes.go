package model

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// SystemAttribute mirrors a row from the public.systemattributes reference
// table: one canonical allowed value for one FISMA system metadata field. The
// accepted vocabulary lives in data rather than code so the frontend builds its
// selects from the backend and writes validate against the same source
// (ztmf#395), the pattern #392 established for datacenterenvironments.
type SystemAttribute struct {
	Field      string `json:"field" db:"field"`
	Value      string `json:"value" db:"value"`
	Selectable bool   `json:"selectable" db:"selectable"`
	Ordr       int    `json:"ordr" db:"ordr"`
}

var systemAttributeColumns = []string{"field", "value", "selectable", "ordr"}

// FindSystemAttributesInput holds optional filters for listing attributes.
type FindSystemAttributesInput struct {
	// Field restricts the result to a single attribute (e.g. "system_type").
	Field *string `schema:"field"`
	// SelectableOnly restricts the result to values offered in the add/edit
	// dropdowns (selectable = TRUE), i.e. hides legacy-alias and field-help rows.
	// Used by the frontend to build the pickers.
	SelectableOnly *bool `schema:"selectable_only"`
}

// FindSystemAttributes returns the systemattributes reference rows. The frontend
// calls it once (optionally with SelectableOnly) and derives every metadata
// select from the result, so the option lists are backend-driven instead of
// hardcoded in the UI.
func FindSystemAttributes(ctx context.Context, input FindSystemAttributesInput) ([]*SystemAttribute, error) {
	sqlb := stmntBuilder.
		Select(systemAttributeColumns...).
		From("public.systemattributes").
		OrderBy("field, ordr, value")

	if input.Field != nil {
		sqlb = sqlb.Where("field = ?", *input.Field)
	}

	if input.SelectableOnly != nil && *input.SelectableOnly {
		sqlb = sqlb.Where("selectable = ?", true)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[SystemAttribute])
}

// systemAttributeValueExists runs a single (field, value) EXISTS check against
// the reference table, used to give a friendly field-level 400 on an off-canon
// enum write before the ztmf#433 CHECK constraint would reject it at the DB.
// Callers validate cloud_service_model element-by-element (the column is a
// text[] since ztmf#433). Both args are bound as parameters. Scoped to
// public.systemattributes; the datacenterenvironments existsIn helper is
// hardcoded to its own table, so this is a separate helper by design.
func systemAttributeValueExists(ctx context.Context, field, value string) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, trapError(err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM public.systemattributes WHERE field = $1 AND value = $2)",
		field, value).Scan(&exists)
	if err != nil {
		return false, trapError(err)
	}
	return exists, nil
}

package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var fismaSystemColumns = []string{"fismasystemid", "fismauid", "fismaacronym", "fismaname", "fismasubsystem", "component", "groupacronym", "groupname", "divisionname", "datacenterenvironment", "datacallcontact", "issoemail", "sdl_sync_enabled", "decommissioned", "decommissioned_date", "decommissioned_by", "decommissioned_notes", "reactivated_by", "reactivated_date", "reactivation_notes", "opdiv_id", "hva", "fips", "system_type", "cloud_system", "cloud_service_model", "cloud_vendor", "system_operator", "goco_coco_gogo", "system_owner", "system_owner_email", "legacy", "isso_name"}

type FismaSystem struct {
	FismaSystemID         int32      `json:"fismasystemid"`
	FismaUID              string     `json:"fismauid"`
	FismaAcronym          string     `json:"fismaacronym"`
	FismaName             string     `json:"fismaname"`
	FismaSubsystem        *string    `json:"fismasubsystem"`
	Component             *string    `json:"component"`
	Groupacronym          *string    `json:"groupacronym"`
	GroupName             *string    `json:"groupname"`
	DivisionName          *string    `json:"divisionname"`
	DataCenterEnvironment *string    `json:"datacenterenvironment"`
	DataCallContact       *string    `json:"datacallcontact"`
	ISSOEmail             *string    `json:"issoemail"`
	SDLSyncEnabled        bool       `json:"sdl_sync_enabled" db:"sdl_sync_enabled"`
	Decommissioned        bool       `json:"decommissioned"`
	DecommissionedDate    *time.Time `json:"decommissioned_date"`
	DecommissionedBy      *string    `json:"decommissioned_by"`
	DecommissionedNotes   *string    `json:"decommissioned_notes"`
	ReactivatedBy         *string    `json:"reactivated_by"`
	ReactivatedDate       *time.Time `json:"reactivated_date"`
	ReactivationNotes     *string    `json:"reactivation_notes"`
	OpDivID               *int32     `json:"opdiv_id" db:"opdiv_id"`
	HVA                   *string    `json:"hva" db:"hva"`
	FIPS                  *string    `json:"fips" db:"fips"`
	SystemType            *string    `json:"system_type" db:"system_type"`
	CloudSystem           *string    `json:"cloud_system" db:"cloud_system"`
	CloudServiceModel     *string    `json:"cloud_service_model" db:"cloud_service_model"`
	CloudVendor           *string    `json:"cloud_vendor" db:"cloud_vendor"`
	SystemOperator        *string    `json:"system_operator" db:"system_operator"`
	GocoCocGoGo           *string    `json:"goco_coco_gogo" db:"goco_coco_gogo"`
	SystemOwner           *string    `json:"system_owner" db:"system_owner"`
	SystemOwnerEmail      *string    `json:"system_owner_email" db:"system_owner_email"`
	Legacy                *string    `json:"legacy" db:"legacy"`
	ISSOName              *string    `json:"isso_name" db:"isso_name"`
}

type FindFismaSystemsInput struct {
	FismaSystemID *int32
	FismaAcronym  *string
	UserID        *string
	OpDivIDs      []int32
	// RestrictToOpDivIDs is the defense-in-depth flag the controller sets
	// when the calling user is an OpDiv-scoped admin. With it set, an empty
	// OpDivIDs slice produces a WHERE FALSE predicate (match no rows) rather
	// than falling through to no-filter. Prevents a fail-open if an
	// OPDIV_ADMIN ends up with zero grants in users_opdivs at any point in
	// their lifecycle (mid-provisioning, all-revoked, etc.).
	RestrictToOpDivIDs bool
	Decommissioned     bool `schema:"decommissioned"`
}

func FindFismaSystems(ctx context.Context, input FindFismaSystemsInput) ([]*FismaSystem, error) {

	c := []string{"fismasystems.fismasystemid as fismasystemid"}
	c = append(c, fismaSystemColumns[1:]...)
	sqlb := stmntBuilder.Select(c...).From("fismasystems")

	// Filter decommissioned systems
	sqlb = sqlb.Where("decommissioned=?", input.Decommissioned)

	// Scope:
	//   - RestrictToOpDivIDs set with an empty slice => fail closed
	//     (WHERE FALSE). Defense-in-depth for the OPDIV_ADMIN-with-no-grants
	//     case so the list endpoint never accidentally returns every row.
	//   - OpDivIDs set => union with system-level assignments via OR. Allows
	//     OpDiv-scoped admins to see every system in their granted OpDivs,
	//     and an ISSO/ISSM who also belongs to an OpDiv to see both their
	//     OpDiv-granted systems and their explicitly-assigned ones.
	//   - UserID set, OpDivIDs empty => legacy behavior, INNER JOIN to
	//     users_fismasystems only (ISSO / ISSM in single-OpDiv state).
	//   - Nothing set => no scope filter (unscoped admin tiers).
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		sqlb = sqlb.Where("FALSE")
	case len(input.OpDivIDs) > 0:
		if input.UserID != nil {
			sqlb = sqlb.Where(
				"(fismasystems.opdiv_id = ANY(?) OR fismasystems.fismasystemid IN (SELECT fismasystemid FROM users_fismasystems WHERE userid = ?))",
				input.OpDivIDs, *input.UserID,
			)
		} else {
			sqlb = sqlb.Where("fismasystems.opdiv_id = ANY(?)", input.OpDivIDs)
		}
	case input.UserID != nil:
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid = fismasystems.fismasystemid AND users_fismasystems.userid=?", *input.UserID)
	}

	if input.FismaAcronym != nil {
		sqlb = sqlb.Where("fismaacronym=?", *input.FismaAcronym)
	}

	sqlb = sqlb.OrderBy("fismasystems.fismasystemid ASC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[FismaSystem])
}

func FindFismaSystem(ctx context.Context, input FindFismaSystemsInput) (*FismaSystem, error) {
	if input.FismaSystemID == nil {
		return nil, &InvalidInputError{
			data: map[string]any{"fismasystemid": nil},
		}
	}

	sqlb := stmntBuilder.
		Select(fismaSystemColumns...).
		From("fismasystems").
		Where("fismasystems.fismasystemid=?", input.FismaSystemID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FismaSystem])
}

// FindFismaSystemByUUID returns the FISMA system identified by its fisma_uuid
// (fismasystems.fismauid, matched case-insensitively), or ErrNoData if none
// matches. Used by the enrichment endpoint to resolve a system's opdiv_id for
// OpDiv-scoped access checks before serving enrichment. fismauid is not unique
// by schema; this returns the first match, which is sufficient for the access
// check since duplicates are not expected in practice. LIMIT 1 makes that
// single-row intent explicit at the query level rather than relying on the
// driver discarding extra rows.
func FindFismaSystemByUUID(ctx context.Context, fismaUUID string) (*FismaSystem, error) {
	if fismaUUID == "" {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.
		Select(fismaSystemColumns...).
		From("fismasystems").
		Where("LOWER(fismauid) = LOWER(?)", fismaUUID).
		Limit(1)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FismaSystem])
}

func (f *FismaSystem) Save(ctx context.Context) (*FismaSystem, error) {

	var sqlb SqlBuilder

	if err := f.validate(); err != nil {
		return nil, err
	}

	// datacenterenvironment is validated against the reference table (ztmf#392)
	// rather than a compiled-in list, so the accepted vocabulary is data. The
	// check lives here rather than in the pure validate() because it needs the
	// request context for the DB lookup.
	if f.DataCenterEnvironment != nil {
		ok, err := dataCenterEnvironmentExists(ctx, *f.DataCenterEnvironment)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &InvalidInputError{data: map[string]any{"datacenterenvironment": *f.DataCenterEnvironment}}
		}
	}

	if f.FismaSystemID == 0 {
		// INSERT - exclude decommissioned/reactivation audit fields. opdiv_id
		// is NOT NULL on the table. Callers may pass an explicit OpDivID; if
		// they do not, default to CMS via subquery so existing CMS admin-panel
		// provisioning keeps working unchanged. HHS OpDiv systems come in via
		// the onboarding workbook importer with OpDivID set explicitly.
		var opdivVal any
		if f.OpDivID != nil {
			opdivVal = *f.OpDivID
		} else {
			opdivVal = squirrel.Expr("(SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS' AND active = TRUE LIMIT 1)")
		}
		insertCols := []string{
			"fismauid", "fismaacronym", "fismaname", "fismasubsystem", "component",
			"groupacronym", "groupname", "divisionname", "datacenterenvironment",
			"datacallcontact", "issoemail", "sdl_sync_enabled", "opdiv_id",
			"hva", "fips", "system_type", "cloud_system", "cloud_service_model",
			"cloud_vendor", "system_operator", "goco_coco_gogo", "system_owner",
			"system_owner_email", "legacy", "isso_name",
		}
		sqlb = stmntBuilder.
			Insert("fismasystems").
			Columns(insertCols...).
			Values(
				f.FismaUID, f.FismaAcronym, f.FismaName, f.FismaSubsystem, f.Component,
				f.Groupacronym, f.GroupName, f.DivisionName, f.DataCenterEnvironment,
				f.DataCallContact, f.ISSOEmail, f.SDLSyncEnabled, opdivVal,
				f.HVA, f.FIPS, f.SystemType, f.CloudSystem, f.CloudServiceModel,
				f.CloudVendor, f.SystemOperator, f.GocoCocGoGo, f.SystemOwner,
				f.SystemOwnerEmail, f.Legacy, f.ISSOName,
			).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))
	} else {
		// UPDATE - exclude decommissioned fields.
		// Core fields are always written; HHS fields are conditional on non-nil
		// so a partial PUT (form that omits a field) does not wipe importer data.
		setCols := squirrel.Eq{
			"fismauid":              f.FismaUID,
			"fismaacronym":          f.FismaAcronym,
			"fismaname":             f.FismaName,
			"fismasubsystem":        f.FismaSubsystem,
			"component":             f.Component,
			"groupacronym":          f.Groupacronym,
			"groupname":             f.GroupName,
			"divisionname":          f.DivisionName,
			"datacenterenvironment": f.DataCenterEnvironment,
			"datacallcontact":       f.DataCallContact,
			"issoemail":             f.ISSOEmail,
			"sdl_sync_enabled":      f.SDLSyncEnabled,
		}
		hhsCols := map[string]*string{
			"hva":                 f.HVA,
			"fips":                f.FIPS,
			"system_type":         f.SystemType,
			"cloud_system":        f.CloudSystem,
			"cloud_service_model": f.CloudServiceModel,
			"cloud_vendor":        f.CloudVendor,
			"system_operator":     f.SystemOperator,
			"goco_coco_gogo":      f.GocoCocGoGo,
			"system_owner":        f.SystemOwner,
			"system_owner_email":  f.SystemOwnerEmail,
			"legacy":              f.Legacy,
			"isso_name":           f.ISSOName,
		}
		for col, val := range hhsCols {
			if val != nil {
				setCols[col] = *val
			}
		}
		sqlb = stmntBuilder.
			Update("fismasystems").
			SetMap(setCols).
			Where("fismasystemid=?", f.FismaSystemID).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FismaSystem])
}

// DecommissionInput contains optional parameters for decommissioning a system
type DecommissionInput struct {
	FismaSystemID      int32
	UserID             string
	DecommissionedDate *time.Time
	Notes              *string
}

// DeleteFismaSystem marks a fismasystem as decommissioned in the database
func DeleteFismaSystem(ctx context.Context, input DecommissionInput) (*FismaSystem, error) {
	if !isValidIntID(input.FismaSystemID) {
		return nil, ErrNoData
	}

	// Validate decommission date is not in future
	if input.DecommissionedDate != nil && input.DecommissionedDate.After(time.Now()) {
		return nil, &InvalidInputError{
			data: map[string]any{"decommissioned_date": "cannot be in the future"},
		}
	}

	sqlb := stmntBuilder.
		Update("fismasystems").
		Set("decommissioned", true).
		Set("decommissioned_by", input.UserID)

	// Use custom date if provided, otherwise NOW()
	if input.DecommissionedDate != nil {
		sqlb = sqlb.Set("decommissioned_date", input.DecommissionedDate)
	} else {
		sqlb = sqlb.Set("decommissioned_date", squirrel.Expr("NOW()"))
	}

	// Add notes if provided
	if input.Notes != nil {
		sqlb = sqlb.Set("decommissioned_notes", *input.Notes)
	}

	sqlb = sqlb.Where("fismasystemid=?", input.FismaSystemID).
		Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FismaSystem])
}

// UpdateDecommissionMetadata allows updating decommission metadata for already-decommissioned systems
func UpdateDecommissionMetadata(ctx context.Context, input DecommissionInput) error {
	if !isValidIntID(input.FismaSystemID) {
		return ErrNoData
	}

	// Validate decommission date is not in future
	if input.DecommissionedDate != nil && input.DecommissionedDate.After(time.Now()) {
		return &InvalidInputError{
			data: map[string]any{"decommissioned_date": "cannot be in the future"},
		}
	}

	// Check if at least one field is provided to update
	if input.DecommissionedDate == nil && input.Notes == nil && input.UserID == "" {
		return &InvalidInputError{
			data: map[string]any{"update": "at least one field must be provided"},
		}
	}

	sqlb := stmntBuilder.
		Update("fismasystems").
		Where("fismasystemid=? AND decommissioned=?", input.FismaSystemID, true)

	// Update date if provided
	if input.DecommissionedDate != nil {
		sqlb = sqlb.Set("decommissioned_date", input.DecommissionedDate)
	}

	// Update notes if provided
	if input.Notes != nil {
		sqlb = sqlb.Set("decommissioned_notes", *input.Notes)
	}

	// Update user if provided
	if input.UserID != "" {
		sqlb = sqlb.Set("decommissioned_by", input.UserID)
	}

	sqlb = sqlb.Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))

	_, err := queryRow(ctx, sqlb, pgx.RowToAddrOfStructByName[FismaSystem])
	return err
}

// ReactivateInput contains parameters for reactivating a decommissioned system
type ReactivateInput struct {
	FismaSystemID int32
	UserID        string
	Notes         *string
}

// ReactivateFismaSystem clears the decommissioned flag and stamps the
// reactivation audit columns. Existing decommissioned_* columns are preserved
// so the prior decommission record remains queryable for history.
//
// Uses a transaction with SELECT FOR UPDATE so the not-found vs already-active
// distinction is decided atomically with the row lock held. Records the audit
// event manually because the transactional path bypasses queryRow's automatic
// recordEvent hook.
func ReactivateFismaSystem(ctx context.Context, input ReactivateInput) (*FismaSystem, error) {
	if !isValidIntID(input.FismaSystemID) {
		return nil, ErrNoData
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, trapError(err)
	}
	// Resolve the transaction and then close the dedicated connection in a single
	// defer, so the order cannot be broken by another defer added later. Rollback
	// is a no-op once the transaction has committed.
	defer func() {
		tx.Rollback(ctx)
		conn.Release()
	}()

	var decommissioned bool
	err = tx.QueryRow(ctx,
		"SELECT decommissioned FROM fismasystems WHERE fismasystemid=$1 FOR UPDATE",
		input.FismaSystemID,
	).Scan(&decommissioned)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoData
	}
	if err != nil {
		return nil, trapError(err)
	}

	if !decommissioned {
		return nil, &InvalidInputError{
			data: map[string]any{"decommissioned": "system is already active"},
		}
	}

	sqlb := stmntBuilder.
		Update("fismasystems").
		Set("decommissioned", false).
		Set("reactivated_by", input.UserID).
		Set("reactivated_date", squirrel.Expr("NOW()"))

	if input.Notes != nil {
		sqlb = sqlb.Set("reactivation_notes", *input.Notes)
	}

	sqlb = sqlb.Where("fismasystemid=?", input.FismaSystemID).
		Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))

	sql, args, err := sqlb.ToSql()
	if err != nil {
		return nil, trapError(err)
	}

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, trapError(err)
	}
	system, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[FismaSystem])
	if err != nil {
		return nil, trapError(err)
	}

	if actor := UserFromContext(ctx); actor != nil {
		if _, err := tx.Exec(ctx,
			"INSERT INTO events (userid, action, resource, payload) VALUES ($1, $2, $3, $4)",
			actor.UserID, "updated", "fismasystems", system,
		); err != nil {
			return nil, trapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, trapError(err)
	}
	return &system, nil
}

func (f *FismaSystem) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if f.FismaUID == "" {
		err.data["fismauid"] = "required"
	}

	if f.DataCallContact != nil && !isValidEmail(*f.DataCallContact) {
		err.data["datacallcontact"] = *f.DataCallContact
	}

	if f.ISSOEmail != nil && !isValidEmail(*f.ISSOEmail) {
		err.data["issoemail"] = *f.ISSOEmail
	}

	// datacenterenvironment is validated against the datacenterenvironments
	// reference table in Save(), which has the context needed for the lookup.

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

package model

import (
	"context"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var fismaSystemColumns = []string{"fismasystemid", "fismauid", "fismaacronym", "fismaname", "fismasubsystem", "component", "groupacronym", "groupname", "divisionname", "datacenterenvironment", "datacallcontact", "issoemail", "sdl_sync_enabled", "decommissioned", "decommissioned_date", "decommissioned_by", "decommissioned_notes"}

type FismaSystem struct {
	FismaSystemID         int32   `json:"fismasystemid"`
	FismaUID              string  `json:"fismauid"`
	FismaAcronym          string  `json:"fismaacronym"`
	FismaName             string  `json:"fismaname"`
	FismaSubsystem        *string `json:"fismasubsystem"`
	Component             *string `json:"component"`
	Groupacronym          *string `json:"groupacronym"`
	GroupName             *string `json:"groupname"`
	DivisionName          *string `json:"divisionname"`
	DataCenterEnvironment *string `json:"datacenterenvironment"`
	DataCallContact       *string `json:"datacallcontact"`
	ISSOEmail             *string    `json:"issoemail"`
	SDLSyncEnabled        bool       `json:"sdl_sync_enabled" db:"sdl_sync_enabled"`
	Decommissioned        bool       `json:"decommissioned"`
	DecommissionedDate    *time.Time `json:"decommissioned_date"`
	DecommissionedBy      *string    `json:"decommissioned_by"`
	DecommissionedNotes   *string    `json:"decommissioned_notes"`
}

type FindFismaSystemsInput struct {
	FismaSystemID  *int32
	FismaAcronym   *string
	UserID         *string
	Decommissioned bool `schema:"decommissioned"`
}

func FindFismaSystems(ctx context.Context, input FindFismaSystemsInput) ([]*FismaSystem, error) {

	c := []string{"fismasystems.fismasystemid as fismasystemid"}
	c = append(c, fismaSystemColumns[1:]...)
	sqlb := stmntBuilder.Select(c...).From("fismasystems")

	// Filter decommissioned systems
	sqlb = sqlb.Where("decommissioned=?", input.Decommissioned)

	if input.UserID != nil {
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

func (f *FismaSystem) Save(ctx context.Context) (*FismaSystem, error) {

	var sqlb SqlBuilder

	if err := f.validate(); err != nil {
		return nil, err
	}

	if f.FismaSystemID == 0 {
		// INSERT - exclude decommissioned fields
		sqlb = stmntBuilder.
			Insert("fismasystems").
			Columns(fismaSystemColumns[1:13]...).
			Values(f.FismaUID, f.FismaAcronym, f.FismaName, f.FismaSubsystem, f.Component, f.Groupacronym, f.GroupName, f.DivisionName, f.DataCenterEnvironment, f.DataCallContact, f.ISSOEmail, f.SDLSyncEnabled).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))
	} else {
		// UPDATE - exclude decommissioned fields
		sqlb = stmntBuilder.Update("fismasystems").
			Set("fismauid", f.FismaUID).
			Set("fismaacronym", f.FismaAcronym).
			Set("fismaname", f.FismaName).
			Set("fismasubsystem", f.FismaSubsystem).
			Set("component", f.Component).
			Set("groupacronym", f.Groupacronym).
			Set("groupname", f.GroupName).
			Set("divisionname", f.DivisionName).
			Set("datacenterenvironment", f.DataCenterEnvironment).
			Set("datacallcontact", f.DataCallContact).
			Set("issoemail", f.ISSOEmail).
			Set("sdl_sync_enabled", f.SDLSyncEnabled).
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

func (f *FismaSystem) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if !isValidUUID(f.FismaUID) {
		err.data["fismauid"] = f.FismaUID
	}

	if f.DataCallContact != nil && !isValidEmail(*f.DataCallContact) {
		err.data["datacallcontact"] = *f.DataCallContact
	}

	if f.ISSOEmail != nil && !isValidEmail(*f.ISSOEmail) {
		err.data["issoemail"] = *f.ISSOEmail
	}

	if f.DataCenterEnvironment != nil && !isValidDataCenterEnvironment(*f.DataCenterEnvironment) {
		err.data["datacenterenvironment"] = *f.DataCenterEnvironment
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

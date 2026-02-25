package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

var cfactsSystemColumns = []string{
	"fisma_uuid",
	"fisma_acronym",
	"authorization_package_name",
	"primary_isso_name",
	"primary_isso_email",
	"is_active",
	"is_retired",
	"is_decommissioned",
	"lifecycle_phase",
	"component_acronym",
	"division_name",
	"group_acronym",
	"group_name",
	"ato_expiration_date",
	"decommission_date",
	"last_modified_date",
	"synced_at",
}

type CfactsSystem struct {
	FismaUUID                string     `json:"fisma_uuid" db:"fisma_uuid"`
	FismaAcronym             string     `json:"fisma_acronym" db:"fisma_acronym"`
	AuthorizationPackageName *string    `json:"authorization_package_name" db:"authorization_package_name"`
	PrimaryISSOName          *string    `json:"primary_isso_name" db:"primary_isso_name"`
	PrimaryISSOEmail         *string    `json:"primary_isso_email" db:"primary_isso_email"`
	IsActive                 *bool      `json:"is_active" db:"is_active"`
	IsRetired                *bool      `json:"is_retired" db:"is_retired"`
	IsDecommissioned         *bool      `json:"is_decommissioned" db:"is_decommissioned"`
	LifecyclePhase           *string    `json:"lifecycle_phase" db:"lifecycle_phase"`
	ComponentAcronym         *string    `json:"component_acronym" db:"component_acronym"`
	DivisionName             *string    `json:"division_name" db:"division_name"`
	GroupAcronym             *string    `json:"group_acronym" db:"group_acronym"`
	GroupName                *string    `json:"group_name" db:"group_name"`
	ATOExpirationDate        *time.Time `json:"ato_expiration_date" db:"ato_expiration_date"`
	DecommissionDate         *time.Time `json:"decommission_date" db:"decommission_date"`
	LastModifiedDate         *time.Time `json:"last_modified_date" db:"last_modified_date"`
	SyncedAt                 time.Time  `json:"synced_at" db:"synced_at"`
}

type FindCfactsSystemsInput struct {
	UserID           *string `schema:"-"`
	FismaAcronym     *string `schema:"fisma_acronym"`
	IsActive         *bool   `schema:"is_active"`
	IsRetired        *bool   `schema:"is_retired"`
	IsDecommissioned *bool   `schema:"is_decommissioned"`
	ComponentAcronym *string `schema:"component_acronym"`
	GroupAcronym     *string `schema:"group_acronym"`
	LifecyclePhase   *string `schema:"lifecycle_phase"`
}

func FindCfactsSystems(ctx context.Context, input FindCfactsSystemsInput) ([]*CfactsSystem, error) {
	// Prefix columns with table name to avoid ambiguity when joining
	cols := make([]string, len(cfactsSystemColumns))
	for i, c := range cfactsSystemColumns {
		cols[i] = "cfacts_systems." + c
	}

	sqlb := stmntBuilder.
		Select(cols...).
		From("cfacts_systems")

	if input.UserID != nil {
		sqlb = sqlb.
			InnerJoin("fismasystems ON fismasystems.fismauid = cfacts_systems.fisma_uuid").
			InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid = fismasystems.fismasystemid AND users_fismasystems.userid = ?", *input.UserID)
	}

	if input.FismaAcronym != nil {
		sqlb = sqlb.Where("cfacts_systems.fisma_acronym=?", *input.FismaAcronym)
	}

	if input.IsActive != nil {
		sqlb = sqlb.Where("cfacts_systems.is_active=?", *input.IsActive)
	}

	if input.IsRetired != nil {
		sqlb = sqlb.Where("cfacts_systems.is_retired=?", *input.IsRetired)
	}

	if input.IsDecommissioned != nil {
		sqlb = sqlb.Where("cfacts_systems.is_decommissioned=?", *input.IsDecommissioned)
	}

	if input.ComponentAcronym != nil {
		sqlb = sqlb.Where("cfacts_systems.component_acronym=?", *input.ComponentAcronym)
	}

	if input.GroupAcronym != nil {
		sqlb = sqlb.Where("cfacts_systems.group_acronym=?", *input.GroupAcronym)
	}

	if input.LifecyclePhase != nil {
		sqlb = sqlb.Where("cfacts_systems.lifecycle_phase=?", *input.LifecyclePhase)
	}

	sqlb = sqlb.OrderBy("cfacts_systems.fisma_acronym ASC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[CfactsSystem])
}

// UserCanAccessCfactsSystem checks whether a user is assigned to the FISMA system
// that corresponds to the given CFACTS fisma_uuid, via the users_fismasystems junction table.
func UserCanAccessCfactsSystem(ctx context.Context, userID string, fismaUUID string) (bool, error) {
	sqlb := stmntBuilder.
		Select("1").
		From("users_fismasystems").
		InnerJoin("fismasystems ON fismasystems.fismasystemid = users_fismasystems.fismasystemid").
		Where("fismasystems.fismauid = ? AND users_fismasystems.userid = ?", fismaUUID, userID).
		Limit(1)

	_, err := queryRow(ctx, sqlb, pgx.RowTo[int])
	if err != nil {
		if err == ErrNoData {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func FindCfactsSystem(ctx context.Context, fismaUUID string) (*CfactsSystem, error) {
	if fismaUUID == "" {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.
		Select(cfactsSystemColumns...).
		From("cfacts_systems").
		Where("fisma_uuid=?", fismaUUID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[CfactsSystem])
}

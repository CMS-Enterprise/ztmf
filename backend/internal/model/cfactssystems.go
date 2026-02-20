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
	FismaAcronym     *string `schema:"fisma_acronym"`
	IsActive         *bool   `schema:"is_active"`
	IsRetired        *bool   `schema:"is_retired"`
	IsDecommissioned *bool   `schema:"is_decommissioned"`
	ComponentAcronym *string `schema:"component_acronym"`
	GroupAcronym     *string `schema:"group_acronym"`
	LifecyclePhase   *string `schema:"lifecycle_phase"`
}

func FindCfactsSystems(ctx context.Context, input FindCfactsSystemsInput) ([]*CfactsSystem, error) {
	sqlb := stmntBuilder.
		Select(cfactsSystemColumns...).
		From("cfacts_systems")

	if input.FismaAcronym != nil {
		sqlb = sqlb.Where("fisma_acronym=?", *input.FismaAcronym)
	}

	if input.IsActive != nil {
		sqlb = sqlb.Where("is_active=?", *input.IsActive)
	}

	if input.IsRetired != nil {
		sqlb = sqlb.Where("is_retired=?", *input.IsRetired)
	}

	if input.IsDecommissioned != nil {
		sqlb = sqlb.Where("is_decommissioned=?", *input.IsDecommissioned)
	}

	if input.ComponentAcronym != nil {
		sqlb = sqlb.Where("component_acronym=?", *input.ComponentAcronym)
	}

	if input.GroupAcronym != nil {
		sqlb = sqlb.Where("group_acronym=?", *input.GroupAcronym)
	}

	if input.LifecyclePhase != nil {
		sqlb = sqlb.Where("lifecycle_phase=?", *input.LifecyclePhase)
	}

	sqlb = sqlb.OrderBy("fisma_acronym ASC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[CfactsSystem])
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

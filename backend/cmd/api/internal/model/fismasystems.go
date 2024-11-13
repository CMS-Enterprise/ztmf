package model

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

var fismaSystemColumns = []string{"fismasystemid", "fismauid", "fismaacronym", "fismaname", "fismasubsystem", "component", "groupacronym", "groupname", "divisionname", "datacenterenvironment", "datacallcontact", "issoemail"}

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
	ISSOEmail             *string `json:"issoemail"`
}

type FindFismaSystemsInput struct {
	FismaSystemID *int32
	FismaAcronym  *string
	UserID        *string
}

func FindFismaSystems(ctx context.Context, input FindFismaSystemsInput) ([]*FismaSystem, error) {

	c := []string{"fismasystems.fismasystemid as fismasystemid"}
	c = append(c, fismaSystemColumns[1:]...)
	sqlb := sqlBuilder.Select(c...).From("fismasystems")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid = fismasystems.fismasystemid AND users_fismasystems.userid=?", *input.UserID)
	}

	if input.FismaAcronym != nil {
		sqlb = sqlb.Where("fismaacronym=?", *input.FismaAcronym)
	}

	sqlb = sqlb.OrderBy("fismasystems.fismasystemid ASC")
	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FismaSystem, error) {
		fismaSystem := FismaSystem{}
		err := row.Scan(&fismaSystem.FismaSystemID, &fismaSystem.FismaUID, &fismaSystem.FismaAcronym, &fismaSystem.FismaName, &fismaSystem.FismaSubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.GroupName, &fismaSystem.DivisionName, &fismaSystem.DataCenterEnvironment, &fismaSystem.DataCallContact, &fismaSystem.ISSOEmail)
		return &fismaSystem, trapError(err)
	})
}

func FindFismaSystem(ctx context.Context, input FindFismaSystemsInput) (*FismaSystem, error) {
	if input.FismaSystemID == nil {
		return nil, &InvalidInputError{
			data: map[string]string{"fismasystemid": "null"},
		}
	}

	sqlb := sqlBuilder.Select(fismaSystemColumns...).From("fismasystems")

	sqlb = sqlb.Where("fismasystems.fismasystemid=?", input.FismaSystemID)
	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	fismaSystem := FismaSystem{}
	err = row.Scan(&fismaSystem.FismaSystemID, &fismaSystem.FismaUID, &fismaSystem.FismaAcronym, &fismaSystem.FismaName, &fismaSystem.FismaSubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.GroupName, &fismaSystem.DivisionName, &fismaSystem.DataCenterEnvironment, &fismaSystem.DataCallContact, &fismaSystem.ISSOEmail)
	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return &fismaSystem, nil
}

func CreateFismaSystem(ctx context.Context, f FismaSystem) (*FismaSystem, error) {
	sqlb := sqlBuilder.
		Insert("fismasystems").
		Columns(fismaSystemColumns[1:]...).
		Values(f.FismaUID, f.FismaAcronym, f.FismaName, f.FismaSubsystem, f.Component, f.Groupacronym, f.GroupName, f.DivisionName, f.DataCenterEnvironment, f.DataCallContact, f.ISSOEmail).
		Suffix("RETURNING fismasystemid")

	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}
	err = row.Scan(&f.FismaSystemID)

	return &f, trapError(err)
}

func UpdateFismaSystem(ctx context.Context, f FismaSystem) (*FismaSystem, error) {
	sqlb := sqlBuilder.Update("fismasystems").
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
		Where("fismasystemid=?", f.FismaSystemID).
		Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))

	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	err = row.Scan(&f.FismaSystemID, &f.FismaUID, &f.FismaAcronym, &f.FismaName, &f.FismaSubsystem, &f.Component, &f.Groupacronym, &f.GroupName, &f.DivisionName, &f.DataCenterEnvironment, &f.DataCallContact, &f.ISSOEmail)

	return &f, trapError(err)
}

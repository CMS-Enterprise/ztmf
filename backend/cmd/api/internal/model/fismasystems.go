package model

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
)

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

	sqlb := sqlBuilder.Select("fismasystems.fismasystemid as fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail").From("fismasystems")

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
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FismaSystem, error) {
		fismaSystem := FismaSystem{}
		err := row.Scan(&fismaSystem.FismaSystemID, &fismaSystem.FismaUID, &fismaSystem.FismaAcronym, &fismaSystem.FismaName, &fismaSystem.FismaSubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.GroupName, &fismaSystem.DivisionName, &fismaSystem.DataCenterEnvironment, &fismaSystem.DataCallContact, &fismaSystem.ISSOEmail)
		return &fismaSystem, err
	})
}

func FindFismaSystem(ctx context.Context, input FindFismaSystemsInput) (*FismaSystem, error) {
	if input.FismaSystemID == nil {
		return nil, errors.New("fismasystemid cannot be null")
	}

	sqlb := sqlBuilder.Select("fismasystems.fismasystemid as fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail").From("fismasystems")

	sqlb = sqlb.Where("fismasystems.fismasystemid=?", input.FismaSystemID)
	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fismaSystem := FismaSystem{}
	err = row.Scan(&fismaSystem.FismaSystemID, &fismaSystem.FismaUID, &fismaSystem.FismaAcronym, &fismaSystem.FismaName, &fismaSystem.FismaSubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.GroupName, &fismaSystem.DivisionName, &fismaSystem.DataCenterEnvironment, &fismaSystem.DataCallContact, &fismaSystem.ISSOEmail)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &fismaSystem, nil
}

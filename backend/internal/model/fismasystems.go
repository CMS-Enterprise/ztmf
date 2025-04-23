package model

import (
	"context"
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
	sqlb := stmntBuilder.Select(c...).From("fismasystems")

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
		sqlb = stmntBuilder.
			Insert("fismasystems").
			Columns(fismaSystemColumns[1:]...).
			Values(f.FismaUID, f.FismaAcronym, f.FismaName, f.FismaSubsystem, f.Component, f.Groupacronym, f.GroupName, f.DivisionName, f.DataCenterEnvironment, f.DataCallContact, f.ISSOEmail).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))
	} else {
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
			Where("fismasystemid=?", f.FismaSystemID).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", "))
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FismaSystem])
}

func (f *FismaSystem) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if !isValidUUID(f.FismaUID) {
		err.data["fismauid"] = f.FismaUID
	}

	if !isValidEmail(*f.DataCallContact) {
		err.data["datacallcontact"] = *f.DataCallContact
	}

	if !isValidEmail(*f.ISSOEmail) {
		err.data["issoemail"] = *f.ISSOEmail
	}

	if !isValidDataCenterEnvironment(*f.DataCenterEnvironment) {
		err.data["datacenterenvironment"] = *f.DataCenterEnvironment
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

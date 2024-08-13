package model

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
)

type FismaSystem struct {
	Fismasystemid         int32
	Fismauid              string
	Fismaacronym          string
	Fismaname             string
	Fismasubsystem        *string
	Component             *string
	Groupacronym          *string
	Groupname             *string
	Divisionname          *string
	Datacenterenvironment *string
	Datacallcontact       *string
	Issoemail             *string
}

type FindFismaSystemsInput struct {
	Fismasystemid *int32
	Fismaacronym  *string
	Userid        *string
}

func FindFismaSystems(ctx context.Context, input FindFismaSystemsInput) ([]*FismaSystem, error) {

	var args = []any{}

	sql := "SELECT fismasystems.fismasystemid as fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail FROM fismasystems"
	if input.Userid != nil {
		sql += joinUserSql(string(*input.Userid))
	}

	if input.Fismaacronym != nil {
		sql += " WHERE fismaacronym=$1"
		args = append(args, input.Fismaacronym)
	}
	sql += " ORDER BY fismasystems.fismasystemid ASC"

	rows, err := query(ctx, sql, args...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FismaSystem, error) {
		fismaSystem := FismaSystem{}
		err := rows.Scan(&fismaSystem.Fismasystemid, &fismaSystem.Fismauid, &fismaSystem.Fismaacronym, &fismaSystem.Fismaname, &fismaSystem.Fismasubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.Groupname, &fismaSystem.Divisionname, &fismaSystem.Datacenterenvironment, &fismaSystem.Datacallcontact, &fismaSystem.Issoemail)
		return &fismaSystem, err
	})
}

func FindFismaSystem(ctx context.Context, input FindFismaSystemsInput) (*FismaSystem, error) {
	if input.Fismasystemid == nil {
		return nil, errors.New("input.Fismasystemid cannot be null")
	}

	var args = []any{}
	sql := "SELECT fismasystems.* FROM fismasystems"
	args = append(args, input.Fismasystemid)

	if input.Userid != nil {
		sql += joinUserSql(*input.Userid)
	}

	sql += " WHERE fismasystems.fismasystemid=$1"
	row, err := queryRow(ctx, sql, args...)
	if err != nil {
		// TODO: make errors more clear where they originated
		log.Println(err)
		return nil, err
	}

	fismaSystem := FismaSystem{}
	err = row.Scan(&fismaSystem.Fismasystemid, &fismaSystem.Fismauid, &fismaSystem.Fismaacronym, &fismaSystem.Fismaname, &fismaSystem.Fismasubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.Groupname, &fismaSystem.Divisionname, &fismaSystem.Datacenterenvironment, &fismaSystem.Datacallcontact, &fismaSystem.Issoemail)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &fismaSystem, nil
}

func joinUserSql(userid string) string {
	return " INNER JOIN users_fismasystems ON users_fismasystems.fismasystemid = fismasystems.fismasystemid AND users_fismasystems.userid = '" + userid + "'"
}

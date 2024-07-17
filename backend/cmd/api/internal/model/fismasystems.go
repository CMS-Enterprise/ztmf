package model

import (
	"context"
	"errors"
	"log"

	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v5"
)

type FismaSystem struct {
	Fismasystemid         graphql.ID
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
	Fismasystemid *graphql.ID
	Fismaacronym  *string
	Userid        *graphql.ID
}

func (f *FismaSystem) FunctionScores(ctx context.Context) ([]*FunctionScore, error) {
	rows, err := query(ctx, "SELECT scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes FROM functionscores WHERE fismasystemid=$1 ORDER BY scoreid ASC", f.Fismasystemid)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FunctionScore, error) {
		functionScore := FunctionScore{}
		err := rows.Scan(&functionScore.Scoreid, &functionScore.Fismasystemid, &functionScore.Functionid, &functionScore.Datecalculated, &functionScore.Score, &functionScore.Notes)
		return &functionScore, err

	})
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
	sql := "SELECT * FROM fismasystems WHERE fismasystemid=$1"
	args = append(args, input.Fismasystemid)

	if input.Userid != nil {
		sql += joinUserSql(string(*input.Userid))
	}

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
	return " JOIN users_fismasystems ON users_fismasystems.fismasystemid = fismasystems.fismasystemid AND users_fismasystems.userid = '" + userid + "'"
}

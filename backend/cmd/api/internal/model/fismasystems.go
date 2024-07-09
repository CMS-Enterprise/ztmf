package model

import (
	"context"
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

func FindFismaSystems(ctx context.Context, fismaacronym *string) ([]*FismaSystem, error) {

	sql := "SELECT * FROM fismasystems"
	if fismaacronym != nil {
		sql += " WHERE fismaacronym=$1"
	}
	sql += " ORDER BY fismasystemid ASC"

	var (
		err  error
		rows pgx.Rows
	)

	if fismaacronym != nil {
		rows, err = query(ctx, sql, fismaacronym)
	} else {
		rows, err = query(ctx, sql)
	}

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

func FindFismaSystemById(ctx context.Context, fismasystemid graphql.ID) (*FismaSystem, error) {

	row, err := queryRow(ctx, "SELECT * FROM fismasystems WHERE fismasystemid=$1", fismasystemid)
	if err != nil {
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

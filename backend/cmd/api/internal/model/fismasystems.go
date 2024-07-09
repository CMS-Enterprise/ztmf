package model

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

type FismaSystem struct {
	Fismasystemid         int
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

func GetFismaSystems(ctx context.Context, fismaacronym string) ([]*FismaSystem, error) {
	db, err := db.Conn(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	sql := "SELECT * FROM fismasystems"
	if fismaacronym != "" {
		sql += " WHERE fismaacronym=$1"
	}
	sql += " ORDER BY fismasystemid ASC"

	var rows pgx.Rows

	if fismaacronym != "" {
		rows, err = db.Query(ctx, sql, fismaacronym)
	} else {
		rows, err = db.Query(ctx, sql)
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

package model

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DataCallFismaSystem struct {
	Datacallid    int32 `json:"datacallid"`
	Fismasystemid int32 `json:"fismasystemid"`
}

func (df *DataCallFismaSystem) Save(ctx context.Context) (*DataCallFismaSystem, error) {
	sqlb := stmntBuilder.
		Insert("datacalls_fismasystems").
		Columns("datacallid", "fismasystemid").
		Values(df.Datacallid, df.Fismasystemid).
		Suffix("ON CONFLICT DO NOTHING RETURNING datacallid, fismasystemid")

	return queryRow(ctx, sqlb, pgx.RowToStructByName[DataCallFismaSystem])
}

// FindDataCallFismaSystems returns all FISMA systems that have marked a specific data call as complete
func FindDataCallFismaSystems(ctx context.Context, datacallID int32) ([]*FismaSystem, error) {
	cols := append(fismaSystemColumns[1:], "fs.fismasystemid")
	sqlb := stmntBuilder.
		Select(cols...).
		From("fismasystems fs").
		InnerJoin("datacalls_fismasystems dcfs ON fs.fismasystemid = dcfs.fismasystemid").
		Where("dcfs.datacallid = ?", datacallID).
		OrderBy("fs.fismaacronym ASC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[FismaSystem])
}

// FindFismaSystemDataCalls returns all data calls that a specific FISMA system has marked as complete
func FindFismaSystemDataCalls(ctx context.Context, fismasystemID int32) ([]*DataCall, error) {
	cols := append(dataCallColumns[1:], "dc.datacallid")
	sqlb := stmntBuilder.
		Select(cols...).
		From("datacalls dc").
		InnerJoin("datacalls_fismasystems dcfs ON dc.datacallid = dcfs.datacallid").
		Where("dcfs.fismasystemid = ?", fismasystemID).
		OrderBy("dc.datecreated DESC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCall])
}

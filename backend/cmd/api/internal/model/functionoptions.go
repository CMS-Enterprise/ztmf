package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type FunctionOption struct {
	FunctionOptionID int32
	FunctionID       int32
	Score            int32
	OptionName       string
	Description      string
}

type FindFunctionOptionsInput struct {
	FunctionID *int32
}

func FindFunctionOptions(ctx context.Context, input FindFunctionOptionsInput) ([]*FunctionOption, error) {
	sqlb := sqlBuilder.Select("functionoptionid,functionid,score,optionname,description").From("functionoptions")

	if input.FunctionID != nil {
		sqlb = sqlb.Where("functionid=?", *input.FunctionID)
	}
	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FunctionOption, error) {
		fo := FunctionOption{}
		err := rows.Scan(&fo.FunctionOptionID, &fo.FunctionID, &fo.Score, &fo.OptionName, &fo.Description)
		return &fo, err
	})
}

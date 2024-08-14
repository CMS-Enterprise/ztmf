package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type FunctionOption struct {
	FunctionOptionID int32  `json:"functionoptionid"`
	FunctionID       int32  `json:"functionid"`
	Score            int32  `json:"score"`
	OptionName       string `json:"optionname"`
	Description      string `json:"description"`
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

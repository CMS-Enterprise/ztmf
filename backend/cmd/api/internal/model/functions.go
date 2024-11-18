package model

import (
	"context"

	"github.com/jackc/pgx/v5"
)

var functionColumns = []string{"functionid", "function", "description", "datacenterenvironment", "questionid", "pillarid"}

type Function struct {
	FunctionID            int32  `json:"functionid"`
	Function              string `json:"function"`
	Description           string `json:"description"`
	DataCenterEnvironment string `json:"datacenterenvironment"`
	QuestionID            int32  `json:"questionid"`
	PillarID              int32  `json:"pillarid"`
}

type FindFunctionsInput struct {
	QuestionID            *int32
	PillarID              *int32
	DataCenterEnvironment *string
}

func FindFunctions(ctx context.Context, i FindFunctionsInput) ([]*Function, error) {
	sqlb := sqlBuilder.
		Select(functionColumns...).
		From("functions")

	if i.QuestionID != nil {
		sqlb = sqlb.Where("questionid=?", *i.QuestionID)
	}

	if i.PillarID != nil {
		sqlb = sqlb.Where("pillarid=?", i.PillarID)
	}

	if i.DataCenterEnvironment != nil {
		sqlb = sqlb.Where("datacenterenvironment=?", i.DataCenterEnvironment)
	}

	sql, boundArgs, _ := sqlb.ToSql()

	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Function, error) {
		f := Function{}
		err := row.Scan(&f.FunctionID, &f.Function, &f.Description, &f.DataCenterEnvironment, &f.QuestionID, &f.PillarID)
		return &f, trapError(err)
	})
}

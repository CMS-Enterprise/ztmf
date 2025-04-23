package model

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

var functionsColumns = []string{"functionid", "function", "description", "datacenterenvironment", "ordr", "questionid", "pillarid"}

type Function struct {
	FunctionID            int32  `json:"functionid"`
	Function              string `json:"function"`
	Description           string `json:"description"`
	DataCenterEnvironment string `json:"datacenterenvironment"`
	Ordr                  int    `json:"order"`
	QuestionID            *int32 `json:"questionid,omitempty"`
	PillarID              int32  `json:"pillarid"`
}

type FindFunctionsInput struct {
	QuestionID            *int32 `schema:"questionid"`
	PillarID              *int32
	DataCenterEnvironment *string
}

func FindFunctions(ctx context.Context, i FindFunctionsInput) ([]*Function, error) {
	sqlb := stmntBuilder.
		Select(functionsColumns...).
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

	sqlb = sqlb.OrderBy("ordr ASC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[Function])
}

// FindFunctionByID queries the database for a Function with the given ID
func FindFunctionByID(ctx context.Context, functionID int32) (*Function, error) {
	if !isValidIntID(functionID) {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.
		Select(functionsColumns...).
		From("functions").
		Where("functionid=?", functionID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[Function])
}

func (f *Function) Save(ctx context.Context) (*Function, error) {

	var sqlb SqlBuilder

	if err := f.validate(); err != nil {
		return nil, err
	}

	if f.FunctionID == 0 {
		sqlb = stmntBuilder.
			Insert("functions").
			Columns(functionsColumns[1:]...).
			Values(f.Function, f.Description, f.DataCenterEnvironment, f.Ordr, f.QuestionID, f.PillarID).
			Suffix("RETURNING " + strings.Join(functionsColumns, ", "))
	} else {
		sqlb = stmntBuilder.Update("functions").
			Set("function", f.Function).
			Set("description", f.Description).
			Set("datacenterenvironment", f.DataCenterEnvironment).
			Set("ordr", f.Ordr).
			Set("questionid", f.QuestionID).
			Set("pillarid", f.PillarID).
			Where("functionid=?", f.FunctionID).
			Suffix("RETURNING " + strings.Join(functionsColumns, ", "))
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByName[Function])
}

func (f *Function) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if f.Function == "" {
		err.data["function"] = ""
	}

	if f.Description == "" {
		err.data["description"] = ""
	}

	if !isValidDataCenterEnvironment(f.DataCenterEnvironment) {
		err.data["datacenterenvironment"] = f.DataCenterEnvironment
	}

	if f.QuestionID != nil && !isValidIntID(f.QuestionID) {
		err.data["questionid"] = f.QuestionID
	}

	if !isValidIntID(f.PillarID) {
		err.data["pillarid"] = f.PillarID
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

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
	Order                 int    `json:"order"`
	QuestionID            *int32 `json:"questionid,omitempty"`
	PillarID              int32  `json:"pillarid"`
}

type FindFunctionsInput struct {
	QuestionID            *int32
	PillarID              *int32
	DataCenterEnvironment *string
}

func FindFunctions(ctx context.Context, i FindFunctionsInput) ([]*Function, error) {
	sqlb := sqlBuilder.
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

	sql, boundArgs, _ := sqlb.ToSql()

	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Function, error) {
		f := Function{}
		err := row.Scan(&f.FunctionID, &f.Function, &f.Description, &f.DataCenterEnvironment, &f.Order, &f.QuestionID, &f.PillarID)
		return &f, trapError(err)
	})
}

// FindFunctionByID queries the database for a Function with the given ID
func FindFunctionByID(ctx context.Context, functionID int32) (*Function, error) {
	if !isValidIntID(functionID) {
		return nil, ErrNoData
	}

	sql, boundArgs, _ := sqlBuilder.
		Select(functionsColumns...).
		From("functions").
		Where("functionid=?", functionID).
		ToSql()

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	// Scan the query result into the User struct
	f := Function{}
	err = row.Scan(&f.FunctionID, &f.Function, &f.Description, &f.DataCenterEnvironment, &f.Order, &f.QuestionID, &f.PillarID)

	return &f, trapError(err)
}

func (f *Function) Save(ctx context.Context) error {

	var (
		sql       string
		boundArgs []any
		err       error
	)

	if valid, err := f.isValid(); !valid {
		return err
	}

	if f.FunctionID == 0 {
		sql, boundArgs, _ = sqlBuilder.
			Insert("functions").
			Columns(functionsColumns[1:]...).
			Values(f.Function, f.Description, f.DataCenterEnvironment, f.Order, f.QuestionID, f.PillarID).
			Suffix("RETURNING " + strings.Join(functionsColumns, ", ")).
			ToSql()
	} else {
		sql, boundArgs, _ = sqlBuilder.Update("functions").
			Set("function", f.Function).
			Set("description", f.Description).
			Set("datacenterenvironment", f.DataCenterEnvironment).
			Set("ordr", f.Order).
			Set("questionid", f.QuestionID).
			Set("pillarid", f.PillarID).
			Where("functionid=?", f.FunctionID).
			Suffix("RETURNING " + strings.Join(functionsColumns, ", ")).
			ToSql()
	}

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return trapError(err)
	}

	err = row.Scan(&f.FunctionID, &f.Function, &f.Description, &f.DataCenterEnvironment, &f.Order, &f.QuestionID, &f.PillarID)

	return trapError(err)
}

func (f *Function) isValid() (isValid bool, e error) {
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

	if len(err.data) == 0 {
		isValid = true
	} else {
		e = &err
	}

	return
}

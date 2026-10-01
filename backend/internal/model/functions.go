package model

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var functionsColumns = []string{"functionid", "function", "description", "datacenterenvironment", "ordr", "questionid", "pillarid"}

type Function struct {
	FunctionID            int32  `json:"functionid"`
	Function              string `json:"function"`
	Description           string `json:"description"`
	DataCenterEnvironment string `json:"datacenterenvironment"`
	// Ordr is a pointer for the same reason Question.Ordr is: a PUT that omits
	// "order" must leave the stored rank alone rather than clobber it to 0.
	// The clobber was harmless while every row was 0; the functions.ordr
	// backfill (ztmf-misc#393) populates this column, so it would now silently
	// destroy the function's position in the questionnaire, the export, and the
	// score diff.
	Ordr       *int   `json:"order"`
	QuestionID *int32 `json:"questionid,omitempty"`
	// Derived from the function's question on write; a value sent by a client is ignored.
	PillarID int32 `json:"pillarid" readonly:"true"`
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

	// functionid breaks ties: after the functions.ordr backfill every edition of
	// a function shares one ordr, so ordr alone would return heap order while
	// looking sorted.
	sqlb = sqlb.OrderBy("ordr ASC, functionid ASC")

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

	// A function's datacenterenvironment must be a scoring key known to the
	// datacenterenvironments mapping (ztmf#392). Checked here rather than in the
	// pure validate() because it needs the request context for the DB lookup.
	ok, err := isValidScoringKey(ctx, f.DataCenterEnvironment)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &InvalidInputError{data: map[string]any{"datacenterenvironment": f.DataCenterEnvironment}}
	}

	// A function's pillar is a fact about its question, so derive it and ignore the
	// caller's value; the caller cannot put functions.pillarid out of agreement with
	// questions.pillarid. validate() has already rejected a nil questionid; the guard
	// repeats it so the deref below does not depend on a check in another method.
	if f.QuestionID == nil {
		return nil, &InvalidInputError{data: map[string]any{"questionid": nil}}
	}

	q, err := FindQuestionByID(ctx, *f.QuestionID)
	if errors.Is(err, ErrNoData) {
		return nil, ErrNoReference
	} else if err != nil {
		return nil, err
	}
	f.PillarID = int32(q.PillarID)

	if f.FunctionID == 0 {
		sqlb = stmntBuilder.
			Insert("functions").
			Columns(functionsColumns[1:]...).
			Values(f.Function, f.Description, f.DataCenterEnvironment, derefInt(f.Ordr), f.QuestionID, f.PillarID).
			Suffix("RETURNING " + strings.Join(functionsColumns, ", "))
	} else {
		ub := stmntBuilder.Update("functions").
			Set("function", f.Function).
			Set("description", f.Description).
			Set("datacenterenvironment", f.DataCenterEnvironment).
			Set("questionid", f.QuestionID).
			Set("pillarid", f.PillarID)
		// Only write ordr when the caller supplied it (see the field comment).
		if f.Ordr != nil {
			ub = ub.Set("ordr", *f.Ordr)
		}
		sqlb = ub.
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

	// datacenterenvironment is validated against the datacenterenvironments
	// reference table in Save(), which has the context needed for the lookup.

	// Required: a function with no question reaches no questionnaire and has no
	// pillar to derive from. pillarid is not validated because Save() overwrites
	// it with the question's pillar.
	if !isValidIntID(f.QuestionID) {
		err.data["questionid"] = f.QuestionID
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

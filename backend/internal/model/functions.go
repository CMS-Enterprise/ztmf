package model

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var functionsColumns = []string{"functionid", "function", "description", "datacenterenvironment", "ordr", "questionid", "pillarid", "versionid", "root_functionid"}

type Function struct {
	FunctionID            int32  `json:"functionid"`
	Function              string `json:"function"`
	Description           string `json:"description"`
	DataCenterEnvironment string `json:"datacenterenvironment"`
	// Rank among the functions of one question. Omitted or null keeps the
	// stored value; a rank cannot be reset to null.
	Ordr       *int   `json:"order"`
	QuestionID *int32 `json:"questionid,omitempty"`
	// Derived from the function's question on write; a value sent by a client is ignored.
	PillarID int32 `json:"pillarid" readonly:"true"`
	// Taken from the function's question (ztmf-misc#394); a value sent by a
	// client is ignored.
	VersionID *int32 `json:"versionid,omitempty" readonly:"true"`
	// The function's identity across versions. Honoured only on create, to
	// restore a lineage (ztmf-misc#396); see Save.
	RootFunctionID *int32 `json:"root_functionid,omitempty" db:"root_functionid"`
}

type FindFunctionsInput struct {
	QuestionID            *int32 `schema:"questionid"`
	PillarID              *int32
	DataCenterEnvironment *string
	// Catalog version to list; nil means the newest published version.
	VersionID *int32 `schema:"versionid"`
}

func FindFunctions(ctx context.Context, i FindFunctionsInput) ([]*Function, error) {
	sqlb := stmntBuilder.
		Select(functionsColumns...).
		From("functions")

	if i.VersionID != nil {
		sqlb = sqlb.Where("versionid=?", *i.VersionID)
	} else {
		sqlb = sqlb.Where(currentQuestionnaireVersionSQL("versionid"))
	}

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

	returning := "RETURNING " + strings.Join(functionsColumns, ", ")

	if f.FunctionID == 0 {
		return catalogWrite(ctx,
			func(pgx.Tx) (int32, error) {
				if q.VersionID == nil {
					return 0, ErrNoReference
				}
				return *q.VersionID, nil
			},
			func(tx pgx.Tx, version int32) error { return f.checkLineage(ctx, tx, version) },
			func(int32) SqlBuilder {
				return stmntBuilder.
					Insert("functions").
					Columns("function", "description", "datacenterenvironment", "ordr", "questionid", "pillarid", "root_functionid").
					Values(f.Function, f.Description, f.DataCenterEnvironment, derefInt(f.Ordr), f.QuestionID, questionPillar(f.QuestionID), f.RootFunctionID).
					Suffix(returning)
			},
			pgx.RowToStructByName[Function], nil)
	}

	return catalogWrite(ctx,
		func(tx pgx.Tx) (int32, error) {
			return storedVersion(ctx, tx, `SELECT versionid FROM functions WHERE functionid = $1`, f.FunctionID)
		},
		func(tx pgx.Tx, _ int32) error { return f.checkLineageKept(ctx, tx) },
		func(int32) SqlBuilder {
			ub := stmntBuilder.Update("functions").
				Set("function", f.Function).
				Set("description", f.Description).
				Set("datacenterenvironment", f.DataCenterEnvironment).
				Set("questionid", f.QuestionID).
				Set("pillarid", questionPillar(f.QuestionID))
			// Only write ordr when the caller supplied it (see the field comment).
			if f.Ordr != nil {
				ub = ub.Set("ordr", *f.Ordr)
			}
			return ub.Where("functionid=?", f.FunctionID).Suffix(returning)
		},
		pgx.RowToStructByName[Function], nil)
}

// questionPillar derives a function's pillar from its question inside the
// write's own statement, so a concurrent pillar change on the question cannot
// leave the function on the old one.
func questionPillar(questionID *int32) squirrel.Sqlizer {
	return squirrel.Expr("(SELECT pillarid FROM questions WHERE questionid = ?)", questionID)
}

// checkLineage decides a new function's lineage (ztmf-misc#396), by the rule
// in lineageRootsSQL: where an earlier version already answers this question
// and environment, the new function must restore one of those lineages; where
// none does, it starts its own and may not name one.
func (f *Function) checkLineage(ctx context.Context, tx pgx.Tx, version int32) error {
	rows, err := tx.Query(ctx, lineageRootsSQL("$1", "$2", "$3")+` ORDER BY 1`, f.QuestionID, f.DataCenterEnvironment, version)
	if err != nil {
		return trapError(err)
	}
	roots, err := pgx.CollectRows(rows, pgx.RowTo[int32])
	if err != nil {
		return trapError(err)
	}
	switch {
	case len(roots) == 0 && f.RootFunctionID == nil:
		return nil
	case len(roots) == 0:
		return &InvalidInputError{data: map[string]any{"root_functionid": "no earlier version has a function for this question and environment to restore"}}
	case f.RootFunctionID == nil || !slices.Contains(roots, *f.RootFunctionID):
		return &InvalidInputError{data: map[string]any{"root_functionid": fmt.Sprintf("an earlier version answers this question and environment under lineage %s; send one as root_functionid to restore it", joinInt32(roots))}}
	}
	return checkRestoredRoot(ctx, tx, "root_functionid", "functions", "functionid", *f.RootFunctionID, version)
}

// checkLineageKept refuses an update that would move a function to a
// different question lineage or environment: both are part of its lineage,
// and moving one would make #395's diff and #397's translation disagree about
// which question an answer belongs to. Delete it and add one instead.
func (f *Function) checkLineageKept(ctx context.Context, tx pgx.Tx) error {
	var questionMoved, environmentMoved bool
	err := tx.QueryRow(ctx, `
		SELECT cq.root_questionid <> nq.root_questionid, cur.datacenterenvironment <> $3
		  FROM functions cur
		 INNER JOIN questions cq ON cq.questionid = cur.questionid
		 INNER JOIN questions nq ON nq.questionid = $2
		 WHERE cur.functionid = $1`, f.FunctionID, f.QuestionID, f.DataCenterEnvironment).Scan(&questionMoved, &environmentMoved)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // a missing row or question fails on the write itself
	}
	if err != nil {
		return trapError(err)
	}
	const why = "is part of a function's lineage; delete it and add a function instead"
	invalid := map[string]any{}
	if questionMoved {
		invalid["questionid"] = "the question " + why
	}
	if environmentMoved {
		invalid["datacenterenvironment"] = "the environment " + why
	}
	if len(invalid) > 0 {
		return &InvalidInputError{data: invalid}
	}
	return nil
}

func joinInt32(s []int32) string {
	parts := make([]string, len(s))
	for i, v := range s {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, " or ")
}

// DeleteFunction removes a function and its options from a draft
// (ztmf-misc#396). Published content is never deleted.
func DeleteFunction(ctx context.Context, functionID int32) error {
	return catalogDelete(ctx, "functions",
		`SELECT versionid FROM functions WHERE functionid = $1`,
		`SELECT versionid FROM functions WHERE functionid = $1 FOR UPDATE`,
		functionSnapshotSQL, functionID,
		func(tx pgx.Tx) error {
			_, err := deleteDraftRow(ctx, tx, "functionid", `DELETE FROM functions WHERE functionid = $1`, functionID)
			return err
		})
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

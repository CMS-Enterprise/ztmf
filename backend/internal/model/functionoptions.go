package model

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	functionOptionColumns = []string{"functionoptions.functionoptionid", "functionid", "score", "optionname", "description"}
)

type FunctionOption struct {
	FunctionOptionID int32 `json:"functionoptionid"`
	// Taken from the path on create and from the stored row on update; a value
	// sent by a client is ignored, so an option cannot be moved between
	// functions by a body field. The tag publishes that as readOnly in the
	// OpenAPI spec; the re-pin in SaveFunctionOption is what enforces it.
	FunctionID  int32  `json:"functionid" readonly:"true"`
	Score       int32  `json:"score"`
	OptionName  string `json:"optionname"`
	Description string `json:"description"`
}

type FindFunctionOptionsInput struct {
	FunctionID *int32
}

func FindFunctionOptions(ctx context.Context, input FindFunctionOptionsInput) ([]*FunctionOption, error) {
	// Maturity order is presentation-critical: the questionnaire renders these
	// rows exactly as returned (the ztmf-ui#369 client-side sort covers pillars
	// and questions, never the answer choices). Without an ORDER BY, Postgres
	// returns heap order, which happened to match score order for years - until
	// the 2026-08-02 description scrub relocated the updated tuples and
	// scrambled the choices on ~50 visible questions across 7 editions, 32
	// minutes before the FY26 call opened (ztmf-misc#279; the audited set is a
	// floor, not a list - heap order reshuffles on any UPDATE or vacuum, which
	// is why the fix lives here and not in the data). score is the natural key
	// (1..4 = traditional..optimal); functionoptionid breaks ties and is known
	// to agree with score order on all 432 functions, so the result is fully
	// deterministic either way.
	sqlb := stmntBuilder.
		Select(functionOptionColumns...).
		From("functionoptions").
		OrderBy("score ASC, functionoptionid ASC")

	if input.FunctionID != nil {
		sqlb = sqlb.Where("functionid=?", *input.FunctionID)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[FunctionOption])
}

// FindFunctionOptionByID queries the database for a FunctionOption with the
// given ID. The update path needs it: PUT /functionoptions/{id} carries no
// functionid in the path, so the stored row is the only trustworthy source for
// which function the option belongs to.
func FindFunctionOptionByID(ctx context.Context, functionOptionID int32) (*FunctionOption, error) {
	if !isValidIntID(functionOptionID) {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.
		Select(functionOptionColumns...).
		From("functionoptions").
		Where("functionoptionid=?", functionOptionID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FunctionOption])
}

// Save creates or updates one answer choice (ztmf-misc#398). Before this the
// table had no write path at all, though the options are the 1-4 scoring scale
// every answer points at, so changing the scale meant hand-written SQL.
//
// There is deliberately no Delete. scores.functionoptionid REFERENCES this
// table with NO ACTION, so deleting an option an answer points at fails on the
// FK rather than orphaning the answer - but an option nothing references yet
// can be deleted, and until the catalog is versioned that is a permanent,
// unversioned, unlogged mutation with no draft to do it in. ztmf-misc#396 adds
// DELETE once a draft version exists that cannot be pinned by a data call, and
// carries the acceptance criterion for it. Do not add it here.
func (fo *FunctionOption) Save(ctx context.Context) (*FunctionOption, error) {

	var sqlb SqlBuilder

	if err := fo.validate(); err != nil {
		return nil, err
	}

	// functionOptionColumns[0] is table-qualified, unlike the other catalog
	// column slices, so RETURNING is built from an unqualified list. Columns
	// [1:] is unaffected either way.
	returning := "functionoptionid, functionid, score, optionname, description"

	if fo.FunctionOptionID == 0 {
		sqlb = stmntBuilder.
			Insert("functionoptions").
			Columns(functionOptionColumns[1:]...).
			Values(fo.FunctionID, fo.Score, fo.OptionName, fo.Description).
			Suffix("RETURNING " + returning)
	} else {
		sqlb = stmntBuilder.Update("functionoptions").
			Set("functionid", fo.FunctionID).
			Set("score", fo.Score).
			Set("optionname", fo.OptionName).
			Set("description", fo.Description).
			Where("functionoptionid=?", fo.FunctionOptionID).
			Suffix("RETURNING " + returning)
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByName[FunctionOption])
}

// maxOptionNameLen and maxOptionDescriptionLen mirror functionoptions'
// varchar(30) and varchar(1024). Checked here so an over-long value is a
// field-named 400 rather than a Postgres string-truncation error (22001, which
// trapError does not map) surfacing as a 500. Both are new reachable inputs -
// the table had no write path before ztmf-misc#398 - so neither inherits the
// unchecked-length gap the other catalog varchars still carry.
const (
	maxOptionNameLen        = 30
	maxOptionDescriptionLen = 1024
)

func (fo *FunctionOption) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if !isValidIntID(fo.FunctionID) {
		err.data["functionid"] = fo.FunctionID
	}

	// 1..4 is the HHS maturity scale (traditional..optimal). Whether a function
	// carries exactly four options with no duplicate score is a publish-time
	// rule, not a write-time one - it belongs to ztmf-misc#396's publish gate,
	// because enforcing it here would make an option impossible to replace
	// while a set is being edited.
	if fo.Score < 1 || fo.Score > 4 {
		err.data["score"] = fo.Score
	}

	// TrimSpace for emptiness, raw length for the bound: " " is a blank answer
	// label, and it is the raw string that has to fit varchar(30).
	if strings.TrimSpace(fo.OptionName) == "" {
		err.data["optionname"] = fo.OptionName
	} else if utf8.RuneCountInString(fo.OptionName) > maxOptionNameLen {
		err.data["optionname"] = fo.OptionName
	}

	// description is nullable in the schema, so unlike Function.Description it
	// is not required - only bounded.
	if utf8.RuneCountInString(fo.Description) > maxOptionDescriptionLen {
		err.data["description"] = fo.Description
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

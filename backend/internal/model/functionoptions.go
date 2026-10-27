package model

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	// Fully qualified because FindScores selects this list across a join to
	// scores, which has carried its own functionid since migration 0060
	// (ztmf#491) - a bare name there is 42702, not a 500 anyone can read.
	functionOptionColumns = []string{
		"functionoptions.functionoptionid",
		"functionoptions.functionid",
		"functionoptions.score",
		"functionoptions.optionname",
		"functionoptions.description",
	}

	// INSERT column lists and RETURNING reject a table-qualified name, so writes
	// use these. Same columns, same order.
	functionOptionWriteColumns = []string{"functionoptionid", "functionid", "score", "optionname", "description"}
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

// Save creates or updates one answer choice (ztmf-misc#398), in a draft
// version only (ztmf-misc#396). The option's version is its function's.
func (fo *FunctionOption) Save(ctx context.Context) (*FunctionOption, error) {
	if err := fo.validate(); err != nil {
		return nil, err
	}
	returning := "RETURNING " + strings.Join(functionOptionWriteColumns, ", ")

	if fo.FunctionOptionID == 0 {
		return catalogWrite(ctx,
			func(tx pgx.Tx) (int32, error) {
				version, err := storedVersion(ctx, tx, `SELECT versionid FROM functions WHERE functionid = $1`, fo.FunctionID)
				if errors.Is(err, ErrNoData) {
					return 0, ErrNoReference
				}
				return version, err
			},
			nil,
			func(int32) SqlBuilder {
				return stmntBuilder.
					Insert("functionoptions").
					Columns(functionOptionWriteColumns[1:]...).
					Values(fo.FunctionID, fo.Score, fo.OptionName, fo.Description).
					Suffix(returning)
			},
			pgx.RowToStructByName[FunctionOption], nil)
	}

	return catalogWrite(ctx,
		func(tx pgx.Tx) (int32, error) {
			return storedVersion(ctx, tx, `SELECT versionid FROM functionoptions WHERE functionoptionid = $1`, fo.FunctionOptionID)
		},
		nil,
		func(int32) SqlBuilder {
			return stmntBuilder.Update("functionoptions").
				Set("functionid", fo.FunctionID).
				Set("score", fo.Score).
				Set("optionname", fo.OptionName).
				Set("description", fo.Description).
				Where("functionoptionid=?", fo.FunctionOptionID).
				Suffix(returning)
		},
		pgx.RowToStructByName[FunctionOption], nil)
}

// DeleteFunctionOption removes an option from a draft (ztmf-misc#396), the
// delete #398 deferred. Safe because no score can reference a draft option,
// which rests on two guards: no data call pins a draft (#397 must refuse a
// draft pin at create and re-pin) and #395's write check refuses an option
// outside the call's pin. A published option is never deleted, which is also
// what keeps undo's prev_functionoptionid resolvable.
func DeleteFunctionOption(ctx context.Context, functionOptionID int32) error {
	return catalogDelete(ctx, "functionoptions",
		`SELECT versionid FROM functionoptions WHERE functionoptionid = $1`,
		`SELECT versionid FROM functionoptions WHERE functionoptionid = $1 FOR UPDATE`,
		optionSnapshotSQL, functionOptionID,
		func(tx pgx.Tx) error {
			_, err := deleteDraftRow(ctx, tx, "functionoptionid", `DELETE FROM functionoptions WHERE functionoptionid = $1`, functionOptionID)
			return err
		})
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

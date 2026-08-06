package model

import (
	"context"

	"github.com/jackc/pgx/v5"
)

var (
	functionOptionColumns = []string{"functionoptions.functionoptionid", "functionid", "score", "optionname", "description"}
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

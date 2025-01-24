package model

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type Answer struct {
	DataCall              string
	FismaSystemID         int32
	FismaAcronym          string
	DataCenterEnvironment string
	Pillar                string
	Question              string
	Function              string
	Description           string
	OptionName            string
	Score                 int
	Notes                 string
}

type FindAnswersInput struct {
	FismaSystemIDs []*int32 `schema:"fsids"`
	DataCallID     int32
	UserID         *string
}

// FindAnswers queries the DB and returns a fully comprehensive set of fields and values
// leveraging all the necessary joins that would otherwise require multiple DB calls
// if using lower-level methods such as FindFismaSystems, FindScores, FindQuestions, etc
// this is primarily meant for use in exporting to spreadsheets
func FindAnswers(ctx context.Context, input FindAnswersInput) ([]*Answer, error) {
	sqlb := stmntBuilder.Select("datacalls.datacall, fismasystems.fismasystemid, fismasystems.fismaacronym, fismasystems.datacenterenvironment, pillars.pillar, questions.question, functions.function, functions.description, functionoptions.optionname, functionoptions.score, scores.notes").
		From("scores").
		InnerJoin("datacalls ON datacalls.datacallid=scores.datacallid AND datacalls.datacallid=?", input.DataCallID).
		InnerJoin("fismasystems ON fismasystems.fismasystemid=scores.fismasystemid").
		InnerJoin("functionoptions ON functionoptions.functionoptionid=scores.functionoptionid").
		InnerJoin("functions ON functions.functionid=functionoptions.functionid").
		InnerJoin("questions ON questions.questionid=functions.questionid").
		InnerJoin("pillars ON pillars.pillarid=functions.pillarid").
		OrderBy("fismasystems.fismasystemid, pillars.ordr, questions.ordr ASC")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.userid=? AND users_fismasystems.fismasystemid=fismasystems.fismasystemid", input.UserID)
	}

	if len(input.FismaSystemIDs) > 0 {
		sqlb = sqlb.Where(squirrel.Eq{"fismasystems.fismasystemid": input.FismaSystemIDs})
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[Answer])

}

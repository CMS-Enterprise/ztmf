package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type Answer struct {
	DataCall              string
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
	FismaSystemIDs []*int32
	DataCallID     int32
	UserID         *string
}

// FindAnswers queries the DB and returns a fully comprehensive set of fields and values
// leveraging all the necessary joins that would otherwise require multiple DB calls
// if using lower-level methods such as FindFismaSystems, FindScores, FindQuestions, etc
// this is primarily meant for use in exporting to spreadsheets
func FindAnswers(ctx context.Context, input FindAnswersInput) ([]*Answer, error) {
	sqlb := sqlBuilder.Select("datacalls.datacall, fismasystems.fismaacronym, fismasystems.datacenterenvironment, pillars.pillar, questions.question, functions.function, functions.description, functionoptions.optionname, functionoptions.score, scores.notes").
		From("scores").
		InnerJoin("datacalls ON datacalls.datacallid=scores.datacallid AND datacalls.datacallid=?", input.DataCallID).
		InnerJoin("fismasystems ON fismasystems.fismasystemid=scores.fismasystemid").
		InnerJoin("functionoptions ON functionoptions.functionoptionid=scores.functionoptionid").
		InnerJoin("functions ON functions.functionid=functionoptions.functionid").
		InnerJoin("questions ON questions.questionid=functions.questionid").
		InnerJoin("pillars ON pillars.pillarid=functions.pillarid").
		OrderBy("pillars.pillar ASC")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.userid=? AND users_fismasystems.fismasystemid=fismasystems.fismasystemid", input.UserID)
	}

	if len(input.FismaSystemIDs) > 0 {
		sqlb = sqlb.Where("fismasystems.fismasystemid IN (1)")
	}

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err, sql)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Answer, error) {
		answer := Answer{}
		err := row.Scan(&answer.DataCall, &answer.FismaAcronym, &answer.DataCenterEnvironment, &answer.Pillar, &answer.Question, &answer.Function, &answer.Description, &answer.OptionName, &answer.Score, &answer.Notes)
		return &answer, trapError(err)
	})
}

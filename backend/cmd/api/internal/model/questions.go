package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type Question struct {
	QuestionID  int32     `json:"questionid"`
	Question    string    `json:"question"`
	Notesprompt string    `json:"notesprompt"`
	Pillar      string    `json:"pillar"`
	Order       int       `json:"order"`
	Function    *Function `json:"function"`
}

type FindQuestionInput struct {
	FismaSystemID *int32
}

func FindQuestions(ctx context.Context, input FindQuestionInput) ([]*Question, error) {
	sqlb := sqlBuilder.Select("questions.questionid, question, notesprompt, pillar, questions.ordr").From("questions").InnerJoin("pillars ON pillars.pillarid=questions.pillarid")

	if input.FismaSystemID != nil {
		sqlb = sqlb.Columns("functionid, function, description").
			InnerJoin("functions ON functions.questionid=questions.questionid").
			InnerJoin("fismasystems ON fismasystems.datacenterenvironment=functions.datacenterenvironment AND fismasystems.fismasystemid=?", *input.FismaSystemID)
	}

	sqlb = sqlb.OrderBy("questions.ordr ASC")

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Question, error) {
		question := Question{}
		scanFields := []any{
			&question.QuestionID,
			&question.Question,
			&question.Notesprompt,
			&question.Pillar,
			&question.Order,
		}

		if input.FismaSystemID != nil {
			question.Function = &Function{}
			scanFields = append(scanFields, &question.Function.FunctionID, &question.Function.Function, &question.Function.Description)
		}

		err := rows.Scan(scanFields...)
		return &question, trapError(err)
	})
}

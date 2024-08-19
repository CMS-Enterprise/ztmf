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
	Function    *Function `json:"function"`
}

type FindQuestionInput struct {
	FismaSystemID *int32
}

func FindQuestions(ctx context.Context, input FindQuestionInput) ([]*Question, error) {
	sqlb := sqlBuilder.Select("questions.questionid, question, notesprompt, pillar").From("questions").InnerJoin("pillars ON pillars.pillarid=questions.pillarid")

	if input.FismaSystemID != nil {
		sqlb = sqlb.Columns("functionid, function, description")
		sqlb = sqlb.InnerJoin("functions ON functions.questionid=questions.questionid")
		sqlb = sqlb.InnerJoin("fismasystems ON fismasystems.datacenterenvironment=functions.datacenterenvironment AND fismasystems.fismasystemid=?", *input.FismaSystemID)
	}

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Question, error) {
		question := Question{}
		scanFields := []any{
			&question.QuestionID,
			&question.Question,
			&question.Notesprompt,
			&question.Pillar,
		}

		if input.FismaSystemID != nil {
			question.Function = &Function{}
			scanFields = append(scanFields, &question.Function.FunctionID, &question.Function.Function, &question.Function.Description)
		}

		err := rows.Scan(scanFields...)
		return &question, err
	})
}

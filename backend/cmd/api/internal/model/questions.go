package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

// Question represents a record from the questions table
type Question struct {
	QuestionID  int32    `json:"questionid"`
	Question    string   `json:"question"`
	Notesprompt string   `json:"notesprompt"`
	Order       int      `json:"order"`
	Pillar      Pillar   `json:"pillar"`
	Function    Function `json:"function"`
}

// FindQuestionsInput provides fields to filter questions by. These are applied in the WHERE clause.
type FindQuestionsInput struct {
	FismaSystemID *int32
}

// FindQuestions performs a select from questions table and returns records as an array of *Questions
func FindQuestions(ctx context.Context, input FindQuestionsInput) ([]*Question, error) {
	sqlb := sqlBuilder.
		Select("questions.questionid, question, notesprompt, questions.ordr, pillars.pillarid, pillars.pillar, pillars.ordr").
		From("questions").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid")

	if input.FismaSystemID != nil {
		sqlb = sqlb.
			Columns("functionid, function, description").
			InnerJoin("functions ON functions.questionid=questions.questionid").
			InnerJoin("fismasystems ON fismasystems.datacenterenvironment=functions.datacenterenvironment AND fismasystems.fismasystemid=?", *input.FismaSystemID)
	}

	sqlb = sqlb.OrderBy("pillars.ordr, questions.ordr ASC")

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
			&question.Order,
			&question.Pillar.PillarID,
			&question.Pillar.Pillar,
			&question.Pillar.Order,
		}

		if input.FismaSystemID != nil {
			question.Function = Function{}
			scanFields = append(scanFields, &question.Function.FunctionID, &question.Function.Function, &question.Function.Description)
		}

		err := rows.Scan(scanFields...)
		return &question, trapError(err)
	})
}

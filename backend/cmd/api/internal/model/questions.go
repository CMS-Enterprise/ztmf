package model

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

var questionsColumns = []string{"questionid", "question", "notesprompt", "ordr", "pillarid"}

// Question represents a record from the questions table
type Question struct {
	QuestionID  int32     `json:"questionid"`
	Question    string    `json:"question"`
	NotesPrompt string    `json:"notesprompt"`
	Ordr        int       `json:"order"`
	PillarID    int       `json:"pillarid"`
	Pillar      *Pillar   `json:"pillar,omitempty"`
	Function    *Function `json:"function,omitempty"`
}

func (q *Question) Save(ctx context.Context) (*Question, error) {

	var sqlb SqlBuilder

	if q.QuestionID == 0 {
		sqlb = stmntBuilder.
			Insert("questions").
			Columns(questionsColumns[1:]...).
			Values(q.Question, q.NotesPrompt, q.Ordr, q.PillarID).
			Suffix("RETURNING " + strings.Join(questionsColumns, ", "))
	} else {
		sqlb = stmntBuilder.Update("questions").
			Set("question", q.Question).
			Set("notesprompt", q.NotesPrompt).
			Set("ordr", q.Ordr).
			Set("pillarid", q.PillarID).
			Where("questionid=?", q.QuestionID).
			Suffix("RETURNING " + strings.Join(questionsColumns, ", "))
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[Question])

}

// func (q *Question) isValid() (bool, error) {
// 	return true, nil
// }

// FindQuestions returns questions without joins, it is used by admins for management
func FindQuestions(ctx context.Context) ([]*Question, error) {
	sqlb := stmntBuilder.
		Select(questionsColumns...).
		From("questions")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByNameLax[Question])
}

func FindQuestionByID(ctx context.Context, questionID int32) (*Question, error) {
	sqlb := stmntBuilder.
		Select(questionsColumns...).
		From("questions").
		Where("questionid=?", questionID)

	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[Question])
}

// FindQuestionsByFismaSystem joins questions with functions to return questions relevant to the fismasystem as determined by the datacenterenvironment.
// It is used by all users to list questions relevant to the specified fisma system
func FindQuestionsByFismaSystem(ctx context.Context, fismaSystemID int32) ([]*Question, error) {
	sqlb := stmntBuilder.
		Select("questions.questionid, question, notesprompt, questions.ordr, pillars.pillarid, pillars.pillar, pillars.ordr, functionid, function, description").
		From("questions").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid").
		InnerJoin("functions ON functions.questionid=questions.questionid").
		InnerJoin("fismasystems ON fismasystems.datacenterenvironment=functions.datacenterenvironment AND fismasystems.fismasystemid=?", fismaSystemID).
		OrderBy("pillars.ordr, questions.ordr ASC")

	return query(ctx, sqlb, func(row pgx.CollectableRow) (*Question, error) {
		q := Question{
			Pillar:   &Pillar{},
			Function: &Function{},
		}
		err := row.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Ordr, &q.Pillar.PillarID, &q.Pillar.Pillar, &q.Pillar.Order, &q.Function.FunctionID, &q.Function.Function, &q.Function.Description)
		return &q, err
	})
}

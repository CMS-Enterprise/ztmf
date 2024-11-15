package model

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

var questionsColumns = []string{"questionid", "question", "notesprompt", "ordr", "pillarid"}

// Question represents a record from the questions table
type Question struct {
	QuestionID  int32     `json:"questionid"`
	Question    string    `json:"question"`
	NotesPrompt string    `json:"notesprompt"`
	Order       int       `json:"order"`
	PillarID    int       `json:"pillarid"`
	Pillar      *Pillar   `json:"pillar,omitempty"`
	Function    *Function `json:"function,omitempty"`
}

func (q *Question) Save(ctx context.Context) error {

	var (
		sql       string
		boundArgs []any
		err       error
	)

	if q.QuestionID == 0 {
		sql, boundArgs, _ = sqlBuilder.
			Insert("questions").
			Columns(questionsColumns[1:]...).
			Values(q.Question, q.NotesPrompt, q.Order, q.PillarID).
			Suffix("RETURNING " + strings.Join(questionsColumns, ", ")).
			ToSql()
	} else {
		sql, boundArgs, _ = sqlBuilder.Update("questions").
			Set("question", q.Question).
			Set("notesprompt", q.NotesPrompt).
			Set("ordr", q.Order).
			Set("pillarid", q.PillarID).
			Where("questionid=?", q.QuestionID).
			Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", ")).
			ToSql()
	}

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return trapError(err)
	}

	err = row.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Order, &q.PillarID)

	return trapError(err)
}

// func (q *Question) isValid() (bool, error) {
// 	return true, nil
// }

// FindQuestions returns questions without joins, it is used by admins for management
func FindQuestions(ctx context.Context) ([]*Question, error) {
	sql, boundArgs, _ := sqlBuilder.
		Select(questionsColumns...).
		From("questions").
		ToSql()

	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Question, error) {
		q := Question{}
		err := rows.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Order, &q.PillarID)
		return &q, trapError(err)
	})

}

// FindQuestionsByFismaSystem joins questions with functions to return questions relevant to the fismasystem as determined by the datacenterenvironment.
// It is used by all users to list questions relevant to the specified fisma system
func FindQuestionsByFismaSystem(ctx context.Context, fismaSystemID int32) ([]*Question, error) {
	sql, boundArgs, _ := sqlBuilder.
		Select("questions.questionid, question, notesprompt, questions.ordr, pillars.pillarid, pillars.pillar, pillars.ordr, functionid, function, description").
		From("questions").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid").
		InnerJoin("functions ON functions.questionid=questions.questionid").
		InnerJoin("fismasystems ON fismasystems.datacenterenvironment=functions.datacenterenvironment AND fismasystems.fismasystemid=?", fismaSystemID).
		OrderBy("pillars.ordr, questions.ordr ASC").
		ToSql()

	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Question, error) {
		q := Question{
			Pillar:   &Pillar{},
			Function: &Function{},
		}
		err := rows.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Order, &q.Pillar.PillarID, &q.Pillar.Pillar, &q.Pillar.Order, &q.Function.FunctionID, &q.Function.Function, &q.Function.Description)
		return &q, trapError(err)
	})
}

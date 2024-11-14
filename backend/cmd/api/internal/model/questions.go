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
	Notesprompt string    `json:"notesprompt"`
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

	err = q.isValid()
	if err != nil {
		return err
	}

	if q.QuestionID == 0 {
		sql, boundArgs, _ = q.insertSql()
	}
	// else {
	// 	sql, boundArgs, _ = q.updateSql()
	// }

	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return trapError(err)
	}

	err = row.Scan(&q.QuestionID, &q.Question, &q.Notesprompt, &q.Order, &q.Pillar.PillarID)

	return trapError(err)
}

func (q *Question) isValid() error {
	return nil
}

func (q *Question) insertSql() (string, []any, error) {
	return sqlBuilder.
		Insert("questions").
		Columns(questionsColumns[1:]...).
		Values().
		Suffix("RETURNING " + strings.Join(questionsColumns, ", ")).
		ToSql()
}

// func (f *) updateSql() (string, []any, error) {
// 	return sqlBuilder.Update("fismasystems").
// 		Set("fismauid", f.FismaUID).
// 		Set("fismaacronym", f.FismaAcronym).
// 		Set("fismaname", f.FismaName).
// 		Set("fismasubsystem", f.FismaSubsystem).
// 		Set("component", f.Component).
// 		Set("groupacronym", f.Groupacronym).
// 		Set("groupname", f.GroupName).
// 		Set("divisionname", f.DivisionName).
// 		Set("datacenterenvironment", f.DataCenterEnvironment).
// 		Set("datacallcontact", f.DataCallContact).
// 		Set("issoemail", f.ISSOEmail).
// 		Where("fismasystemid=?", f.FismaSystemID).
// 		Suffix("RETURNING " + strings.Join(fismaSystemColumns, ", ")).
// 		ToSql()

// }

// FindQuestions returns questions
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
		err := rows.Scan(&q.QuestionID, &q.Question, &q.Notesprompt, &q.Order, &q.PillarID)
		return &q, trapError(err)
	})

}

// FindQuestionsByFismaSystem joins questions with functions to return questions relevant to the fismasystem as determined by the datacenterenvironment
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
		err := rows.Scan(&q.QuestionID, &q.Question, &q.Notesprompt, &q.Order, &q.Pillar.PillarID, &q.Pillar.Pillar, &q.Pillar.Order, &q.Function.FunctionID, &q.Function.Function, &q.Function.Description)
		return &q, trapError(err)
	})
}

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
	// Ordr is a pointer so a PUT that omits "order" leaves the stored rank
	// alone instead of clobbering it to 0. Before migration 0056 populated
	// these values the clobber was a harmless no-op; now it would silently
	// destroy a question's canonical position in the questionnaire, the
	// export, and the score diff (the CLAUDE.md optional-field rule - it
	// applies to every optional column, not just bools).
	Ordr        *int      `json:"order"`
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
			Values(q.Question, q.NotesPrompt, derefInt(q.Ordr), q.PillarID).
			Suffix("RETURNING " + strings.Join(questionsColumns, ", "))
	} else {
		ub := stmntBuilder.Update("questions").
			Set("question", q.Question).
			Set("notesprompt", q.NotesPrompt).
			Set("pillarid", q.PillarID)
		// Only write ordr when the caller supplied it (see the field comment).
		if q.Ordr != nil {
			ub = ub.Set("ordr", *q.Ordr)
		}
		sqlb = ub.
			Where("questionid=?", q.QuestionID).
			Suffix("RETURNING " + strings.Join(questionsColumns, ", "))
	}

	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[Question])

}

// func (q *Question) validate() error {
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

// FindQuestionsByFismaSystemInput carries the optional query filters for the
// nested questions listing.
type FindQuestionsByFismaSystemInput struct {
	// DataCallID applies the reduced-pillar rule for the cycle being answered
	// (ztmf#545): questions on a pillar a seeded rule excludes for this
	// system's environment are omitted. Absent, the full catalog for the
	// environment is returned, which is the pre-#545 contract the frontend's
	// own pillar filter still expects.
	DataCallID *int32 `schema:"datacallid"`
}

// FindQuestionsByFismaSystem joins questions with functions to return questions relevant to the fismasystem as determined by the datacenterenvironment.
// It is used by all users to list questions relevant to the specified fisma system
func FindQuestionsByFismaSystem(ctx context.Context, fismaSystemID int32, input FindQuestionsByFismaSystemInput) ([]*Question, error) {
	// The system's raw datacenterenvironment is resolved to a scoring vocabulary
	// through the datacenterenvironments mapping (ztmf#392); functions are matched
	// on that key. This is the same indirection the scoring aggregate uses, so the
	// answer form an ISSO sees matches the functions a system is scored against.
	sqlb := stmntBuilder.
		Select("questions.questionid, question, notesprompt, questions.ordr, pillars.pillarid, pillars.pillar, pillars.ordr, functionid, function, description").
		From("questions").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid").
		InnerJoin("functions ON functions.questionid=questions.questionid").
		InnerJoin("datacenterenvironments dce ON dce.scoring_key=functions.datacenterenvironment").
		InnerJoin("fismasystems ON fismasystems.datacenterenvironment=dce.datacenterenvironment AND fismasystems.fismasystemid=?", fismaSystemID).
		// questionid breaks ties so questions sharing an ordr (0 wherever
		// migration 0056 found no canonical rank) still list deterministically
		// rather than in heap order. See FindAnswers for the same tiebreaker.
		OrderBy("pillars.ordr, questions.ordr, questions.questionid ASC")

	if input.DataCallID != nil {
		sqlb = sqlb.Where(reducedPillarScopeSQL("dce.scoring_key", "pillars.pillar", "?"), *input.DataCallID)
	}

	return query(ctx, sqlb, func(row pgx.CollectableRow) (*Question, error) {
		q := Question{
			Pillar:   &Pillar{},
			Function: &Function{},
		}
		err := row.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Ordr, &q.Pillar.PillarID, &q.Pillar.Pillar, &q.Pillar.Order, &q.Function.FunctionID, &q.Function.Function, &q.Function.Description)
		return &q, err
	})
}

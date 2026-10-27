package model

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

var questionsColumns = []string{"questionid", "question", "notesprompt", "ordr", "pillarid", "versionid", "root_questionid"}

// Question represents a record from the questions table
type Question struct {
	QuestionID  int32     `json:"questionid"`
	Question    string    `json:"question"`
	NotesPrompt string    `json:"notesprompt"`
	// Rank within the question's pillar. Omitted or null keeps the stored
	// value; a rank cannot be reset to null.
	Ordr        *int      `json:"order"`
	PillarID    int       `json:"pillarid"`
	// The questionnaire version this row belongs to (ztmf-misc#394). Honoured
	// only on create, where it must name a draft; omitted, the open draft is
	// used. Fixed after create.
	VersionID *int32 `json:"versionid,omitempty"`
	// The question's identity across versions. Honoured only on create, to
	// restore a lineage a draft deleted (ztmf-misc#396); omitted, a new
	// question starts its own.
	RootQuestionID *int32  `json:"root_questionid,omitempty" db:"root_questionid"`
	Pillar         *Pillar `json:"pillar,omitempty"`
	Function    *Function `json:"function,omitempty"`
}

func (q *Question) Save(ctx context.Context) (*Question, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}
	returning := "RETURNING " + strings.Join(questionsColumns, ", ")

	if q.QuestionID == 0 {
		return catalogWrite(ctx,
			func(tx pgx.Tx) (int32, error) {
				if q.VersionID != nil {
					return *q.VersionID, nil
				}
				return openDraftVersionID(ctx, tx)
			},
			func(tx pgx.Tx, version int32) error {
				if q.RootQuestionID == nil {
					return nil
				}
				return checkRestoredRoot(ctx, tx, "root_questionid", "questions", "questionid", *q.RootQuestionID, version)
			},
			func(version int32) SqlBuilder {
				return stmntBuilder.
					Insert("questions").
					Columns("question", "notesprompt", "ordr", "pillarid", "versionid", "root_questionid").
					Values(q.Question, q.NotesPrompt, derefInt(q.Ordr), q.PillarID, version, q.RootQuestionID).
					Suffix(returning)
			},
			pgx.RowToStructByNameLax[Question], nil)
	}

	return catalogWrite(ctx,
		func(tx pgx.Tx) (int32, error) {
			return storedVersion(ctx, tx, `SELECT versionid FROM questions WHERE questionid = $1`, q.QuestionID)
		},
		nil,
		func(int32) SqlBuilder {
			ub := stmntBuilder.Update("questions").
				Set("question", q.Question).
				Set("notesprompt", q.NotesPrompt).
				Set("pillarid", q.PillarID)
			// Only write ordr when the caller supplied it (see the field comment).
			if q.Ordr != nil {
				ub = ub.Set("ordr", *q.Ordr)
			}
			return ub.Where("questionid=?", q.QuestionID).Suffix(returning)
		},
		pgx.RowToStructByNameLax[Question],
		// A function's pillar is its question's (Function.Save derives it), so a
		// pillar change carries to the question's functions rather than leaving
		// them for the publish gate to refuse.
		func(tx pgx.Tx, saved *Question) error {
			rows, err := tx.Query(ctx, `UPDATE functions SET pillarid = $1 WHERE questionid = $2 AND pillarid <> $1 RETURNING functionid`, saved.PillarID, saved.QuestionID)
			if err != nil {
				return trapError(err)
			}
			moved, err := pgx.CollectRows(rows, pgx.RowTo[int32])
			if err != nil || len(moved) == 0 {
				return trapError(err)
			}
			return insertCatalogEvent(ctx, tx, eventActionUpdated, "functions",
				map[string]any{"questionid": saved.QuestionID, "pillarid": saved.PillarID, "functionids": moved})
		})
}

// DeleteQuestion removes a question, and its functions and their options, from
// a draft (ztmf-misc#396). Published content is never deleted. The question
// row is locked before its functions, the same order a pillar change takes.
func DeleteQuestion(ctx context.Context, questionID int32) error {
	return catalogDelete(ctx, "questions",
		`SELECT versionid FROM questions WHERE questionid = $1`,
		`SELECT versionid FROM questions WHERE questionid = $1 FOR UPDATE`,
		questionSnapshotSQL, questionID,
		func(tx pgx.Tx) error {
			// functions.questionid does not cascade; options cascade from functions.
			if _, err := deleteDraftRow(ctx, tx, "questionid", `DELETE FROM functions WHERE questionid = $1`, questionID); err != nil {
				return err
			}
			_, err := deleteDraftRow(ctx, tx, "questionid", `DELETE FROM questions WHERE questionid = $1`, questionID)
			return err
		})
}

// validate rejects a question the questions table would not accept, or would
// accept as meaningless. question, notesprompt and pillarid are all NOT NULL
// with no useful default, so all three are required on create and on update -
// Save's UPDATE branch sets each unconditionally, so a PUT that omits one would
// otherwise clobber it to the zero value rather than leave it alone.
//
// ordr is deliberately not required: it is a *int precisely so a PUT that omits
// "order" preserves the stored rank (see the field comment).
//
// Pillar existence is left to the FK. questions.pillarid REFERENCES pillars, so
// a well-formed but non-existent id raises 23503, which trapError already maps
// to ErrNoReference and a 400 - the same status this returns, just without a
// field name. Not an oversight: pillars carries no finder to pre-empt it with,
// unlike Function.Save, which had to read its question anyway to derive pillarid.
func (q *Question) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	// TrimSpace, not == "": a question of " " is NOT NULL and non-empty as far
	// as the column is concerned, but renders blank in the questionnaire and in
	// the export. Rejected rather than trimmed - silently rewriting a caller's
	// text is worse than refusing it.
	if strings.TrimSpace(q.Question) == "" {
		err.data["question"] = q.Question
	}

	if strings.TrimSpace(q.NotesPrompt) == "" {
		err.data["notesprompt"] = q.NotesPrompt
	}

	// Not isValidIntID: that helper takes any but type-switches only int32 and
	// *int32, so a plain int falls through and reports false for every value.
	if q.PillarID <= 0 {
		err.data["pillarid"] = q.PillarID
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

// FindQuestionsInput selects the catalog version to list; nil means the newest
// published version.
type FindQuestionsInput struct {
	VersionID *int32 `schema:"versionid"`
}

// FindQuestions returns questions without joins, it is used by admins for management
func FindQuestions(ctx context.Context, input FindQuestionsInput) ([]*Question, error) {
	sqlb := stmntBuilder.
		Select(questionsColumns...).
		From("questions")

	if input.VersionID != nil {
		sqlb = sqlb.Where("versionid=?", *input.VersionID)
	} else {
		sqlb = sqlb.Where(currentQuestionnaireVersionSQL("versionid"))
	}

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
	// DataCallID resolves the cycle's pinned questionnaire version
	// (ztmf-misc#395) and applies its reduced-pillar rule (ztmf#545): questions
	// on a pillar a seeded rule excludes for this system's environment are
	// omitted. Absent, the environment's questions in the newest published
	// version are returned, with no pillar rule - so a client reading an older
	// cycle must pass it.
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
		Select("questions.questionid, question, notesprompt, questions.ordr, pillars.pillarid, pillars.pillar, pillars.ordr, functionid, function, description, functions.ordr").
		From("questions").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid").
		InnerJoin("functions ON functions.questionid=questions.questionid").
		InnerJoin("datacenterenvironments dce ON dce.scoring_key=functions.datacenterenvironment").
		InnerJoin("fismasystems ON fismasystems.datacenterenvironment=dce.datacenterenvironment AND fismasystems.fismasystemid=?", fismaSystemID).
		// functions.ordr orders the functions a single question fans out to; the
		// catalog gives each (question, environment) one row, so today it only
		// matters for a question that forks. questionid breaks the remaining ties
		// so rows an ordr backfill could not rank still list deterministically
		// rather than in heap order. See FindAnswers for the same tiebreaker.
		OrderBy("pillars.ordr, questions.ordr, functions.ordr, questions.questionid ASC")

	// Without a data call there is no pin, so the newest published version is
	// the catalog; leaving it unscoped would list every version's questions.
	if input.DataCallID != nil {
		sqlb = sqlb.
			Where(reducedPillarScopeSQL("dce.scoring_key", "pillars.pillar", "?"), *input.DataCallID).
			Where(questionnaireVersionSQL("functions.versionid", "?"), *input.DataCallID)
	} else {
		sqlb = sqlb.Where(currentQuestionnaireVersionSQL("functions.versionid"))
	}

	return query(ctx, sqlb, func(row pgx.CollectableRow) (*Question, error) {
		q := Question{
			Pillar:   &Pillar{},
			Function: &Function{},
		}
		err := row.Scan(&q.QuestionID, &q.Question, &q.NotesPrompt, &q.Ordr, &q.Pillar.PillarID, &q.Pillar.Pillar, &q.Pillar.Order, &q.Function.FunctionID, &q.Function.Function, &q.Function.Description, &q.Function.Ordr)
		return &q, err
	})
}

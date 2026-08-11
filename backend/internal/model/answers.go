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
	// The selected option and its notes are nil for an applicable function the
	// system has not answered in this data call: FindAnswers left joins scores
	// off the questionnaire, so an unanswered function still yields a row but
	// carries no score. The export renders these as blank, which distinguishes a
	// never-answered function from one genuinely answered at score 0.
	OptionDescription     *string
	OptionName            *string
	Score                 *int
	Notes                 *string
	NotesIsAISummary      *bool `db:"notes_is_ai_summary"`
	// Per-system target maturity (#398); nil when no target has been asserted
	// (the export renders the Advanced default in that case).
	TargetMaturityTier          *string `db:"target_maturity_tier"`
	TargetMaturityJustification *string `db:"target_maturity_justification"`
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
	sqlb := stmntBuilder.Select("datacalls.datacall, fismasystems.fismasystemid, fismasystems.fismaacronym, fismasystems.datacenterenvironment, fismasystems.target_maturity_tier, fismasystems.target_maturity_justification, pillars.pillar, questions.question, functions.function, functions.description, functionoptions.description AS optiondescription, functionoptions.optionname, functionoptions.score, scores.notes, scores.notes_is_ai_summary").
		From("fismasystems").
		InnerJoin("functions ON (EXISTS (SELECT 1 FROM datacenterenvironments dce WHERE dce.datacenterenvironment=fismasystems.datacenterenvironment AND dce.scoring_key=functions.datacenterenvironment) OR EXISTS (SELECT 1 FROM scores answered JOIN functionoptions answeredopt ON answeredopt.functionoptionid=answered.functionoptionid WHERE answered.fismasystemid=fismasystems.fismasystemid AND answered.datacallid=? AND answeredopt.functionid=functions.functionid))", input.DataCallID).
		InnerJoin("questions ON questions.questionid=functions.questionid").
		InnerJoin("pillars ON pillars.pillarid=questions.pillarid").
		InnerJoin("datacalls ON datacalls.datacallid=?", input.DataCallID).
		LeftJoin("scores ON scores.fismasystemid=fismasystems.fismasystemid AND scores.datacallid=? AND scores.functionoptionid IN (SELECT selected.functionoptionid FROM functionoptions selected WHERE selected.functionid=functions.functionid)", input.DataCallID).
		LeftJoin("functionoptions ON functionoptions.functionoptionid=scores.functionoptionid").
		Where("(fismasystems.decommissioned=FALSE OR scores.scoreid IS NOT NULL)").
		// Top-level WHERE, not a condition on the functions join: that join is
		// applicable-OR-answered (#528), so filtering one branch would export 40
		// rows for a system with carried-forward excluded answers and 25 for a
		// fresh one. The scoring key needs a subquery because no dce alias is in
		// scope here.
		Where(
			saasPillarScopeSQL(
				"(SELECT dcescope.scoring_key FROM datacenterenvironments dcescope WHERE dcescope.datacenterenvironment=fismasystems.datacenterenvironment)",
				"pillars.pillar",
				"?",
			),
			input.DataCallID,
		).
		// questionid is the tiebreaker, not decoration: pillars.ordr and
		// questions.ordr are 0 for any row migration 0056 could not rank (the
		// empire seed's fictional function names), and rows tied on the sort key
		// would otherwise come back in heap order, which shifts whenever a row is
		// rewritten. questions.questionid alone does not reach the export's row grain: the
	// applicable-or-answered join deliberately admits functions from OTHER
	// editions when a system answered them before an environment change
	// (#528), so one questionid can yield two rows that tie on every
	// question-level column. functions.functionid settles those, keeping the
	// export byte-stable.
		OrderBy("fismasystems.fismasystemid, pillars.ordr, questions.ordr, questions.questionid, functions.functionid ASC")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.userid=? AND users_fismasystems.fismasystemid=fismasystems.fismasystemid", input.UserID)
	}

	if len(input.FismaSystemIDs) > 0 {
		sqlb = sqlb.Where(squirrel.Eq{"fismasystems.fismasystemid": input.FismaSystemIDs})
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[Answer])

}

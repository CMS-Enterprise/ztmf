package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ScoreProgress is one FISMA system's questionnaire progress within a single
// data call, built for the "which systems have not updated their
// questionnaires" dashboard view.
//
// QuestionsUpdated counts questions genuinely touched this cycle, not
// questions that merely have an answer row: when a data call is created,
// copyPreviousScores pre-populates the previous cycle's answers WITHOUT
// recording events, so a carried-forward untouched answer has no event row.
// Counting rows would therefore read ~100% for every carried-over system;
// counting rows WITH events is what distinguishes real updates.
type ScoreProgress struct {
	FismaSystemID int32 `json:"fismasystemid"`
	// QuestionsExpected is the number of questionnaire functions applicable to
	// the system, resolved through the datacenterenvironments scoring
	// vocabulary (the same join the questionnaire and the score aggregation
	// use, so the denominator matches what an ISSO actually sees).
	QuestionsExpected int32 `json:"questionsexpected"`
	// QuestionsUpdated is the number of distinct functions whose answer in
	// this data call has at least one recorded edit event.
	QuestionsUpdated int32 `json:"questionsupdated"`
	// LastUpdatedAt is the most recent edit event across the system's answers
	// in this data call; nil when nothing has been touched this cycle.
	LastUpdatedAt *time.Time `json:"lastupdatedat,omitempty"`
	// UpdatedSinceStart is a convenience flag: true when at least one answer
	// has been edited in this data call.
	UpdatedSinceStart bool `json:"updatedsincestart"`
}

type FindScoreProgressInput struct {
	DataCallID    *int32 `schema:"datacallid"`
	FismaSystemID *int32 `schema:"fismasystemid"`
	// UserID restricts progress to the requesting user's assigned systems
	// (ISSO/ISSM tiers); set by the controller, not bindable from the query.
	UserID *string
	// OpDiv scope for OpDiv-scoped admin tiers, mirroring FindScores. Not
	// schema-tagged - the controller sets them from the auth'd user.
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

func (i FindScoreProgressInput) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if i.DataCallID == nil {
		err.data["datacallid"] = "required"
	}

	if len(err.data) > 0 {
		return &err
	}
	return nil
}

// FindScoreProgress returns one row per in-scope FISMA system with its
// questionnaire progress for the given data call: how many questions apply to
// the system, how many were genuinely updated this cycle, and when the most
// recent update happened.
//
// Systems the caller cannot see are excluded by the same scoping rules as
// FindScores (per-user assigned systems, OpDiv grants, or unrestricted).
// Decommissioned systems are not filtered here - the caller joins progress
// onto whatever system list it displays, so extra rows are inert.
//
// The query is hand-built parameterized SQL through the read-only rawQuery
// path (never queryRow, which records events), mirroring FindScoreDiff. The
// events lateral is the same shape FindScores uses for last_edited_at and is
// served by the events_score_audit_idx partial index.
func FindScoreProgress(ctx context.Context, input FindScoreProgressInput) ([]*ScoreProgress, error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	sql, args := buildScoreProgressSQL(input)
	return query(ctx, rawQuery{sql: sql, args: args}, scanScoreProgress)
}

// buildScoreProgressSQL assembles the parameterized progress query. Extracted
// so unit tests can pin the filter and scope shaping without a database
// connection. validate() guarantees DataCallID is non-nil before this is
// called.
func buildScoreProgressSQL(input FindScoreProgressInput) (string, []any) {
	var args []any
	argN := 1

	// System-scope predicates applied to the expected CTE, which anchors the
	// result set: a system outside the caller's scope produces no row at all,
	// and a system inside scope with zero activity still produces a row (that
	// zero-activity row is the entire point of the feature).
	conds := []string{"TRUE"}

	if input.FismaSystemID != nil {
		conds = append(conds, fmt.Sprintf("fs.fismasystemid = $%d", argN))
		args = append(args, *input.FismaSystemID)
		argN++
	}

	if input.UserID != nil {
		conds = append(conds, fmt.Sprintf("fs.fismasystemid IN (SELECT fismasystemid FROM users_fismasystems WHERE userid = $%d)", argN))
		args = append(args, *input.UserID)
		argN++
	}

	// OpDiv scope (fail-closed): empty grants under RestrictToOpDivIDs -> no
	// rows, matching FindScores.
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		conds = append(conds, "FALSE")
	case len(input.OpDivIDs) > 0:
		conds = append(conds, fmt.Sprintf("fs.opdiv_id = ANY($%d)", argN))
		args = append(args, input.OpDivIDs)
		argN++
	}

	dataCallArg := argN
	args = append(args, *input.DataCallID)
	argN++

	// expected: functions applicable to each in-scope system via the
	// datacenterenvironments scoring-vocabulary mapping (same indirection as
	// FindQuestionsByFismaSystem). LEFT joins so a system whose environment
	// has no mapping still returns with an expected count of zero.
	//
	// updated: distinct functions whose score row in this data call has at
	// least one edit event. The INNER lateral join is the filter - a
	// pre-populated row copied by copyPreviousScores has no event and drops
	// out here, which is exactly the "has a row but was not updated" case.
	sql := fmt.Sprintf(`
WITH expected AS (
    SELECT fs.fismasystemid, COUNT(f.functionid) AS questionsexpected
    FROM fismasystems fs
    LEFT JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
    LEFT JOIN functions f ON f.datacenterenvironment = dce.scoring_key
    WHERE %s
    GROUP BY fs.fismasystemid
),
updated AS (
    SELECT s.fismasystemid,
           COUNT(DISTINCT fo.functionid) AS questionsupdated,
           MAX(le.createdat) AS lastupdatedat
    FROM scores s
    INNER JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
    INNER JOIN LATERAL (
        SELECT createdat
        FROM events
        WHERE resource = 'public.scores'
          AND (payload->>'scoreid')::int = s.scoreid
        ORDER BY createdat DESC
        LIMIT 1
    ) le ON TRUE
    WHERE s.datacallid = $%d
    GROUP BY s.fismasystemid
)
SELECT ex.fismasystemid,
       ex.questionsexpected,
       COALESCE(u.questionsupdated, 0) AS questionsupdated,
       u.lastupdatedat
FROM expected ex
LEFT JOIN updated u ON u.fismasystemid = ex.fismasystemid
ORDER BY ex.fismasystemid
`, strings.Join(conds, " AND "), dataCallArg)

	return sql, args
}

func scanScoreProgress(row pgx.CollectableRow) (*ScoreProgress, error) {
	var p ScoreProgress

	if err := row.Scan(&p.FismaSystemID, &p.QuestionsExpected, &p.QuestionsUpdated, &p.LastUpdatedAt); err != nil {
		return nil, err
	}

	p.UpdatedSinceStart = p.QuestionsUpdated > 0

	return &p, nil
}

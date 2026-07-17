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
// It carries two distinct numerators over the same applicable-function
// denominator, because the dashboard asks two different questions depending on
// whether the data call is active or historical:
//
//   - QuestionsUpdated ("touched THIS cycle") drives the active-call progress
//     chip. A carried-forward answer does not count until re-saved (ztmf#299).
//   - QuestionsAnswered ("has an answer at all") is the completion signal a
//     PAST call needs. A closed cycle stops accumulating updates, and history
//     imported from outside the app never had any (no events to backfill), so
//     QuestionsUpdated says nothing about whether a historical call was in
//     fact complete - QuestionsAnswered does (ztmf-ui#537). Cycles genuinely
//     worked in-app keep their backfilled updated counts; accurate history,
//     just not a completion signal.
//
// Both are now read from persisted state (scores.status and score-row
// presence) rather than reconstructed from the events audit log; see
// FindScoreProgress.
type ScoreProgress struct {
	FismaSystemID int32 `json:"fismasystemid"`
	// QuestionsExpected is the number of questionnaire functions applicable to
	// the system, resolved through the datacenterenvironments scoring
	// vocabulary (the same join the questionnaire and the score aggregation
	// use, so the denominator matches what an ISSO actually sees).
	QuestionsExpected int32 `json:"questionsexpected"`
	// QuestionsAnswered is the number of distinct applicable functions that
	// have an answer row in this data call, regardless of whether it was
	// touched this cycle. Counted from the same applicable-function set as
	// QuestionsExpected, so it can never exceed it. This is the answered/total
	// count a historical (closed) data call reports - carried-forward answers
	// count here even though they do not count as QuestionsUpdated.
	QuestionsAnswered int32 `json:"questionsanswered"`
	// QuestionsUpdated is the number of distinct functions whose answer in
	// this data call reached status = 'done' (genuinely saved this cycle) AND
	// is still applicable to the system's current environment. Counted from
	// the same applicable-function set as QuestionsExpected, so it can never
	// exceed it. Read from the persisted scores.status column, not from the
	// events audit log.
	QuestionsUpdated int32 `json:"questionsupdated"`
	// LastUpdatedAt is the most recent edit event across the system's answers
	// in this data call; nil when nothing has been touched this cycle. This is
	// a legitimately observational use of the events table (audit timeline),
	// kept even though the counts no longer read events.
	LastUpdatedAt *time.Time `json:"lastupdatedat,omitempty"`
	// UpdatedSinceStart is derivable (QuestionsUpdated > 0) but kept because
	// it answers the ticket's literal question - "has this system updated
	// since the start of the data call" - by name in the response, so a
	// consumer rendering a boolean chip never touches the numeric fields.
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
// Decommissioned systems are excluded: they do not participate in data
// calls, so a "not updated" row for one is pure noise in the triage view
// this endpoint feeds. If a consumer ever needs historical progress for a
// decommissioned system, add an opt-in query param rather than removing
// the filter.
//
// The query is hand-built parameterized SQL through the read-only rawQuery
// path (never queryRow, which records events), mirroring FindScoreDiff. The
// counts derive from persisted state - score-row presence for answered,
// scores.status for updated - so a dropped/failed event write no longer moves
// the numbers. A LEFT lateral onto events remains only for LastUpdatedAt (an
// audit timeline, not a count) and is served by the events_score_audit_idx
// partial index.
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

	// System-scope predicates on the fismasystems anchor. Applied once in the
	// scoped_systems CTE that both expected and updated read from, so updated
	// never computes progress for a system the caller cannot see and a
	// single-system request only touches that system's rows. Decommissioned
	// systems are out of scope for data call participation entirely.
	conds := []string{"fs.decommissioned = FALSE"}

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

	// All count halves draw from the SAME applicable-function set - the exact
	// set FindQuestionsByFismaSystem resolves for the questionnaire an ISSO
	// fills out: functions with a question (INNER, so orphan functions are
	// excluded), a valid pillar, matched to the system's environment through
	// the datacenterenvironments scoring vocabulary. answered's and updated's
	// sets are that same set further restricted to answered functions (and, for
	// updated, to status = 'done'), so both are subsets of expected's - neither
	// questionsanswered nor questionsupdated can exceed questionsexpected even
	// when a carried-over answer references a function that is no longer
	// applicable after an environment change (that answer simply fails the
	// applicability join). COUNT(DISTINCT) on all guards against fan-out from
	// the environment mapping.
	sql := fmt.Sprintf(`
WITH scoped_systems AS (
    SELECT fs.fismasystemid, fs.datacenterenvironment
    FROM fismasystems fs
    WHERE %s
),
expected AS (
    SELECT ss.fismasystemid, COUNT(DISTINCT f.functionid) AS questionsexpected
    FROM scoped_systems ss
    INNER JOIN datacenterenvironments dce ON dce.datacenterenvironment = ss.datacenterenvironment
    INNER JOIN functions f ON f.datacenterenvironment = dce.scoring_key
    INNER JOIN questions q ON q.questionid = f.questionid
    INNER JOIN pillars p ON p.pillarid = q.pillarid
    GROUP BY ss.fismasystemid
),
updated AS (
    SELECT ss.fismasystemid,
           -- every applicable answered function (answered/total for a closed
           -- call); carried-forward rows count here.
           COUNT(DISTINCT f.functionid) AS questionsanswered,
           -- only those genuinely saved THIS cycle. status is a persisted
           -- fact written in the same statement as the answer, so a
           -- pre-populated row copied by copyPreviousScores (status =
           -- 'not_started') is excluded without consulting the events log.
           COUNT(DISTINCT f.functionid) FILTER (WHERE s.status = 'done') AS questionsupdated,
           MAX(le.createdat) AS lastupdatedat -- newest across the system's rows; the lateral below is per-row
    FROM scoped_systems ss
    INNER JOIN scores s ON s.fismasystemid = ss.fismasystemid AND s.datacallid = $%d
    INNER JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
    INNER JOIN functions f ON f.functionid = fo.functionid
    INNER JOIN datacenterenvironments dce ON dce.datacenterenvironment = ss.datacenterenvironment
                                         AND dce.scoring_key = f.datacenterenvironment
    INNER JOIN questions q ON q.questionid = f.questionid
    INNER JOIN pillars p ON p.pillarid = q.pillarid
    -- One newest event per score row (LIMIT 1 keeps the lateral on the
    -- index fast path); the outer MAX then picks the newest across the
    -- system's rows. LEFT now (not the old filtering INNER): it feeds only
    -- LastUpdatedAt, an audit timeline - a row with no event still counts
    -- toward the numerators via status/presence and simply contributes no
    -- timestamp.
    LEFT JOIN LATERAL (
        SELECT createdat
        FROM events
        WHERE resource = 'public.scores'
          AND (payload->>'scoreid')::int = s.scoreid
        ORDER BY createdat DESC
        LIMIT 1
    ) le ON TRUE
    GROUP BY ss.fismasystemid
)
SELECT ss.fismasystemid,
       COALESCE(ex.questionsexpected, 0) AS questionsexpected,
       COALESCE(u.questionsanswered, 0) AS questionsanswered,
       COALESCE(u.questionsupdated, 0) AS questionsupdated,
       u.lastupdatedat
FROM scoped_systems ss
LEFT JOIN expected ex ON ex.fismasystemid = ss.fismasystemid
LEFT JOIN updated u ON u.fismasystemid = ss.fismasystemid
ORDER BY ss.fismasystemid
`, strings.Join(conds, " AND "), dataCallArg)

	return sql, args
}

func scanScoreProgress(row pgx.CollectableRow) (*ScoreProgress, error) {
	var p ScoreProgress

	if err := row.Scan(&p.FismaSystemID, &p.QuestionsExpected, &p.QuestionsAnswered, &p.QuestionsUpdated, &p.LastUpdatedAt); err != nil {
		return nil, err
	}

	p.UpdatedSinceStart = p.QuestionsUpdated > 0

	return &p, nil
}

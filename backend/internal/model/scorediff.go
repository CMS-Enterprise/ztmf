package model

import (
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/jackc/pgx/v5"
)

// ScoreDiffSide is one cycle's answer to a single questionnaire function: the
// selected functionoption (with its maturity score + label) and any notes.
// One side may be nil on a ScoreDiff when that function was answered in only
// one of the two data calls (a newly-answered or no-longer-answered function).
type ScoreDiffSide struct {
	ScoreID          int32   `json:"scoreid"`
	FunctionOptionID int32   `json:"functionoptionid"`
	OptionName       string  `json:"optionname"`
	Score            int32   `json:"score"`
	Notes            *string `json:"notes"`
	NotesIsAISummary *bool   `json:"notes_is_ai_summary"`
}

// ScoreDiff is a single questionnaire function whose answer changed between two
// data calls for one FISMA system. From is the answer in the earlier ("from")
// cycle and To the answer in the later ("to") cycle; either may be nil when the
// function was answered in only one cycle. ChangedAt/ChangedBy attribute the
// write that produced the To answer (the change), resolved from the events
// audit trail; both are nil when no write event exists for the To score (e.g.
// seed data, or a function answered only in the From cycle).
type ScoreDiff struct {
	FismaSystemID int32          `json:"fismasystemid"`
	FunctionID    int32          `json:"functionid"`
	Function      string         `json:"function"`
	Question      string         `json:"question"`
	From          *ScoreDiffSide `json:"from"`
	To            *ScoreDiffSide `json:"to"`
	ChangedAt     *time.Time     `json:"changed_at,omitempty"`
	ChangedBy     *AuditRef      `json:"changed_by,omitempty"`
}

type FindScoreDiffInput struct {
	FismaSystemID  *int32 `schema:"fismasystemid"`
	FromDataCallID *int32 `schema:"from"`
	ToDataCallID   *int32 `schema:"to"`
	// UserID restricts the diff to the requesting user's assigned systems
	// (ISSO/ISSM tiers); set by the controller, not bindable from the query.
	UserID *string
	// OpDiv scope for OpDiv-scoped admin tiers, mirroring FindScores. Not
	// schema-tagged - the controller sets them from the auth'd user.
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

// FindScoreDiff compares the score (functionoption) answers of two data calls
// for the in-scope FISMA system(s) and returns only the functions whose answer
// differs between them, each annotated with who made the later change and when.
//
// "Differs" means the selected functionoption changed, the notes changed, or
// the function was answered in exactly one of the two cycles. Notes are
// compared with whitespace normalized (nil and empty treated as equal, leading
// and trailing whitespace trimmed, internal runs collapsed) so neither a
// NULL-vs-"" notes box nor a spacing-only difference surfaces as a spurious
// change (see #409).
//
// The query is hand-built parameterized SQL rather than squirrel because the
// FULL OUTER JOIN between the two cycles, the catalog joins for the function /
// question labels, and the events lateral for attribution exceed squirrel's
// ergonomics. It is SELECT-only, so it goes through the read-path query helper
// (never queryRow, which records events). See [[scores.FindScores]] for the
// attribution lateral this mirrors.
func FindScoreDiff(ctx context.Context, input FindScoreDiffInput) ([]*ScoreDiff, error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	sql, args := buildScoreDiffSQL(input)
	return query(ctx, rawQuery{sql: sql, args: args}, scanScoreDiff)
}

func (i FindScoreDiffInput) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if i.FromDataCallID == nil {
		err.data["from"] = "required"
	}
	if i.ToDataCallID == nil {
		err.data["to"] = "required"
	}
	if i.FromDataCallID != nil && i.ToDataCallID != nil && *i.FromDataCallID == *i.ToDataCallID {
		err.data["to"] = "must differ from 'from'"
	}

	if len(err.data) > 0 {
		return &err
	}
	return nil
}

// buildScoreDiffSQL assembles the parameterized diff query. Extracted so unit
// tests can pin the filter and scope shaping without a database connection.
// validate() guarantees both data call IDs are non-nil before this is called.
func buildScoreDiffSQL(input FindScoreDiffInput) (string, []any) {
	var args []any
	argN := 1

	fromCTE := scoreCycleCTE(input, *input.FromDataCallID, &args, &argN)
	toCTE := scoreCycleCTE(input, *input.ToDataCallID, &args, &argN)

	// FULL OUTER JOIN on (system, function) so a function answered in only one
	// cycle still surfaces (added or removed answer). Catalog joins are LEFT so
	// a row whose function/question was since deleted still returns with empty
	// labels rather than vanishing. The events lateral keys on the To scoreid
	// because the later write is what "who made the change" refers to; the
	// partial index events_score_audit_idx serves it. The IS DISTINCT FROM
	// pair is the "drop the unchanged rows" filter.
	//
	// Notes are compared with whitespace normalized (leading/trailing trimmed
	// and internal runs collapsed to a single space) so a note that differs
	// only in spacing -- e.g. the FY23 two-spaces-after-a-period vintage vs
	// FY24 single spaces -- is not reported as a change. Only the comparison is
	// normalized; the stored notes are returned verbatim (see #409).
	sql := fmt.Sprintf(`
WITH from_scores AS (%s),
     to_scores AS (%s)
SELECT
    COALESCE(f.fismasystemid, t.fismasystemid) AS fismasystemid,
    COALESCE(f.functionid, t.functionid)       AS functionid,
    fn.function,
    q.question,
    f.scoreid, f.functionoptionid, f.option_score, f.optionname, f.notes, f.notes_is_ai_summary,
    t.scoreid, t.functionoptionid, t.option_score, t.optionname, t.notes, t.notes_is_ai_summary,
    le.createdat,
    eu.userid, eu.fullname, eu.email, eu.role
FROM from_scores f
FULL OUTER JOIN to_scores t
  ON f.fismasystemid = t.fismasystemid AND f.functionid = t.functionid
LEFT JOIN functions fn ON fn.functionid = COALESCE(f.functionid, t.functionid)
LEFT JOIN questions q  ON q.questionid  = fn.questionid
LEFT JOIN LATERAL (
    SELECT createdat, userid
    FROM events
    WHERE resource = 'public.scores'
      AND (payload->>'scoreid')::int = t.scoreid
    ORDER BY createdat DESC
    LIMIT 1
) le ON TRUE
LEFT JOIN users eu ON eu.userid = le.userid
WHERE f.functionoptionid IS DISTINCT FROM t.functionoptionid
   OR btrim(regexp_replace(COALESCE(f.notes, ''), '\s+', ' ', 'g'))
      IS DISTINCT FROM
      btrim(regexp_replace(COALESCE(t.notes, ''), '\s+', ' ', 'g'))
   OR f.notes_is_ai_summary IS DISTINCT FROM t.notes_is_ai_summary
ORDER BY fismasystemid, fn.ordr, functionid
`, fromCTE, toCTE)

	return sql, args
}

// scoreCycleCTE builds the SELECT for one data call's scored functions, joined
// to functionoptions for the maturity score + label and scoped identically to
// FindScores (per-user assigned systems, OpDiv grants, or unrestricted). It
// appends its bound values to args in placeholder order; both the from and to
// cycles call it so the scope values are bound once per cycle.
func scoreCycleCTE(input FindScoreDiffInput, dataCallID int32, args *[]any, argN *int) string {
	var userJoin string
	if input.UserID != nil {
		userJoin = fmt.Sprintf("INNER JOIN users_fismasystems ufs ON ufs.fismasystemid = s.fismasystemid AND ufs.userid = $%d", *argN)
		*args = append(*args, *input.UserID)
		*argN++
	}

	conds := []string{fmt.Sprintf("s.datacallid = $%d", *argN)}
	*args = append(*args, dataCallID)
	*argN++

	if input.FismaSystemID != nil {
		conds = append(conds, fmt.Sprintf("s.fismasystemid = $%d", *argN))
		*args = append(*args, *input.FismaSystemID)
		*argN++
	}

	// OpDiv scope (fail-closed): empty grants under RestrictToOpDivIDs -> no
	// rows. Expressed as a subquery predicate to match FindScores.
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		conds = append(conds, "FALSE")
	case len(input.OpDivIDs) > 0:
		conds = append(conds, fmt.Sprintf("s.fismasystemid IN (SELECT fismasystemid FROM fismasystems WHERE opdiv_id = ANY($%d))", *argN))
		*args = append(*args, input.OpDivIDs)
		*argN++
	}

	return fmt.Sprintf(`
    SELECT s.scoreid, s.fismasystemid, fo.functionid, s.functionoptionid,
           fo.score AS option_score, fo.optionname, s.notes, s.notes_is_ai_summary
    FROM scores s
    INNER JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
    %s
    WHERE %s`, userJoin, strings.Join(conds, " AND "))
}

func scanScoreDiff(row pgx.CollectableRow) (*ScoreDiff, error) {
	var (
		d  ScoreDiff
		fn *string
		qn *string

		fScoreID    *int32
		fFOID       *int32
		fOptScore   *int32
		fOptName    *string
		fNotes      *string
		fAISummary  *bool

		tScoreID    *int32
		tFOID       *int32
		tOptScore   *int32
		tOptName    *string
		tNotes      *string
		tAISummary  *bool

		at     *time.Time
		uID    *string
		uName  *string
		uEmail *string
		uRole  *string
	)

	if err := row.Scan(
		&d.FismaSystemID, &d.FunctionID, &fn, &qn,
		&fScoreID, &fFOID, &fOptScore, &fOptName, &fNotes, &fAISummary,
		&tScoreID, &tFOID, &tOptScore, &tOptName, &tNotes, &tAISummary,
		&at, &uID, &uName, &uEmail, &uRole,
	); err != nil {
		return nil, err
	}

	d.Function = derefString(fn)
	d.Question = derefString(qn)

	// scoreid is NOT NULL in the scores table, so a non-nil scoreid is the
	// signal that this side of the FULL OUTER JOIN matched a row; the other
	// answer columns on that side are then guaranteed populated.
	if fScoreID != nil {
		d.From = &ScoreDiffSide{
			ScoreID:          *fScoreID,
			FunctionOptionID: derefInt32(fFOID),
			OptionName:       derefString(fOptName),
			Score:            derefInt32(fOptScore),
			Notes:            fNotes,
			NotesIsAISummary: fAISummary,
		}
	}
	if tScoreID != nil {
		d.To = &ScoreDiffSide{
			ScoreID:          *tScoreID,
			FunctionOptionID: derefInt32(tFOID),
			OptionName:       derefString(tOptName),
			Score:            derefInt32(tOptScore),
			Notes:            tNotes,
			NotesIsAISummary: tAISummary,
		}
	}

	// Both-or-neither, mirroring FindScores: only attribute when the event
	// timestamp AND the editor identity both resolved.
	if at != nil && uID != nil {
		d.ChangedAt = at
		d.ChangedBy = &AuditRef{
			UserID: *uID,
			Name:   derefString(uName),
			Email:  derefString(uEmail),
			Role:   derefString(uRole),
		}
	}

	return &d, nil
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

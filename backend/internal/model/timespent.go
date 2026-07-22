package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// idleCapSeconds bounds the dwell attributed to a single question view. A view
// with no following event within this window (the user walked away, closed the
// tab, or left it open overnight) is clamped to this value rather than counting
// the full wall-clock gap. Tunable here without touching stored data; it is a
// compile-time constant, never client input, so it is safe to inline into the
// query string.
const idleCapSeconds = 20 * 60

// TimeSpent is one FISMA system's questionnaire effort within a data call,
// derived from the 'viewed' events the questionnaire records. "Time spent" on a
// question is the interval from its 'viewed' event to the next event by the same
// user in the same system+data call (the next view, or a save), clamped at
// idleCapSeconds. Each view's readonly flag splits the dwell into editor
// (readonly=false) or viewer (readonly=true) time.
//
// Time is only counted from when view tracking was enabled: a data call with no
// 'viewed' pings (e.g. one that predates the feature, or one nobody has opened
// since) yields no rows. The numbers are a lower bound, not stopwatch-exact:
// every dwell is capped at the idle limit, and the last event a user makes
// before leaving has no trailing event so it contributes nothing. They answer
// the FY2026 question - roughly how long people spend - not billing-grade
// timekeeping.
type TimeSpent struct {
	FismaSystemID int32 `json:"fismasystemid"`
	// EditorSeconds / ViewerSeconds split TotalSeconds by the mode of each
	// contributing view: editor activity (views made while editing) versus
	// read-only viewing.
	EditorSeconds float64 `json:"editor_seconds"`
	ViewerSeconds float64 `json:"viewer_seconds"`
	// TotalSeconds is the summed capped dwell (editor + viewer) across every
	// person and question for this system in the data call.
	TotalSeconds float64 `json:"total_seconds"`
	// QuestionsMeasured is the count of distinct questions that accrued any
	// dwell for this system; the denominator for AverageSecondsPerQuestion.
	QuestionsMeasured int32 `json:"questions_measured"`
	// AverageSecondsPerQuestion is TotalSeconds / QuestionsMeasured (0 when no
	// question was measured) - the issue's "average time spent per question
	// within 1 system".
	AverageSecondsPerQuestion float64 `json:"average_seconds_per_question"`
	// PerPerson breaks the system's total down by person, so a consumer can see
	// time spent per person for each system.
	PerPerson []*TimeSpentPerson `json:"per_person"`
	// PerQuestion breaks the system's total down by question.
	PerQuestion []*TimeSpentQuestion `json:"per_question"`
}

// TimeSpentPerson is one person's contribution to a system's effort, split into
// editor and viewer time.
type TimeSpentPerson struct {
	AuditRef
	EditorSeconds     float64 `json:"editor_seconds"`
	ViewerSeconds     float64 `json:"viewer_seconds"`
	TotalSeconds      float64 `json:"total_seconds"`
	QuestionsMeasured int32   `json:"questions_measured"`
}

// TimeSpentQuestion is one question's effort within a system: the average time
// each contributing person spent on it, and how many distinct people did. This
// is a per-person average (question total / distinct people), distinct from
// TimeSpent.AverageSecondsPerQuestion, which is the system's total / question
// count.
type TimeSpentQuestion struct {
	QuestionID              int32   `json:"questionid"`
	AverageSecondsPerPerson float64 `json:"average_seconds_per_person"`
	People                  int32   `json:"people"`
}

// timeSpentRow is the (system, user, question) grain the dwell query returns;
// FindTimeSpent rolls these up into per-system TimeSpent values. Seconds is
// pre-split into editor and viewer buckets by the query so the rollup stays a
// dumb accumulator.
type timeSpentRow struct {
	FismaSystemID int32
	UserID        string
	Name          string
	Email         string
	Role          string
	QuestionID    int32
	EditorSeconds float64
	ViewerSeconds float64
}

func (r *timeSpentRow) seconds() float64 { return r.EditorSeconds + r.ViewerSeconds }

type FindTimeSpentInput struct {
	DataCallID    *int32 `schema:"datacallid"`
	FismaSystemID *int32 `schema:"fismasystemid"`
	// UserID restricts results to the requesting user's assigned systems
	// (ISSO/ISSM tiers); set by the controller, not bindable from the query.
	UserID *string
	// OpDiv scope for OpDiv-scoped admin tiers, mirroring FindScoreProgress. Not
	// schema-tagged - the controller sets them from the auth'd user.
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

func (i FindTimeSpentInput) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if i.DataCallID == nil {
		err.data["datacallid"] = "required"
	}

	if len(err.data) > 0 {
		return &err
	}
	return nil
}

// FindTimeSpent returns questionnaire effort per in-scope FISMA system for the
// given data call: total time split into editor/viewer buckets, a per-person
// breakdown, a per-question breakdown, and the average per question. Systems the
// caller cannot see are excluded by the same scoping rules as FindScoreProgress
// (per-user assigned systems, OpDiv grants, or unrestricted).
//
// The query goes through the read-only rawQuery path (never queryRow, which
// records events), mirroring FindScoreProgress.
func FindTimeSpent(ctx context.Context, input FindTimeSpentInput) ([]*TimeSpent, error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	sql, args := buildTimeSpentSQL(input)
	rows, err := query(ctx, rawQuery{sql: sql, args: args}, scanTimeSpentRow)
	if err != nil {
		return nil, err
	}
	return rollupTimeSpent(rows), nil
}

// timeSpentScope builds the scoped_systems predicate and returns the WHERE
// fragment, the accumulated args, and the placeholder index carrying the data
// call id (appended last).
//
// System-scope predicates apply once on the fismasystems anchor, so effort is
// never computed for a system the caller cannot see. Decommissioned systems are
// intentionally NOT excluded here (unlike the progress triage view): this is
// historical analytics and effort spent on a system before it was decommissioned
// is still real.
func timeSpentScope(input FindTimeSpentInput) (where string, args []any, dataCallArg int) {
	argN := 1
	var conds []string

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
	// rows, matching FindScoreProgress.
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		conds = append(conds, "FALSE")
	case len(input.OpDivIDs) > 0:
		conds = append(conds, fmt.Sprintf("fs.opdiv_id = ANY($%d)", argN))
		args = append(args, input.OpDivIDs)
		argN++
	}

	where = "TRUE"
	if len(conds) > 0 {
		where = strings.Join(conds, " AND ")
	}

	dataCallArg = argN
	args = append(args, *input.DataCallID)
	return where, args, dataCallArg
}

// buildTimeSpentSQL assembles the parameterized dwell query at the
// (system, user, question) grain. Extracted so unit tests can pin the filter and
// scope shaping without a database connection. validate() guarantees DataCallID
// is non-nil before this is called.
//
// stream is the per-(user, system) ordered event timeline over BOTH the 'viewed'
// pings and the 'public.scores' saves, so LEAD gives each view the timestamp of
// the next thing the user did (moved to another question, or saved). Only
// 'viewed' rows enter dwell - saves are boundary markers only, which is why their
// payload lacking a questionid is irrelevant. The dwell is clamped at the idle
// cap so an abandoned/overnight view cannot balloon a total. The view's readonly
// flag routes its dwell into the editor or viewer bucket. idleCapSeconds is a
// compile-time constant, so inlining it is safe.
func buildTimeSpentSQL(input FindTimeSpentInput) (string, []any) {
	where, args, dataCallArg := timeSpentScope(input)

	sql := fmt.Sprintf(`
WITH scoped_systems AS (
    SELECT fs.fismasystemid
    FROM fismasystems fs
    WHERE %s
),
stream AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'questionid')::int    AS questionid,
           (e.payload->>'readonly')::boolean  AS readonly,
           e.action,
           e.createdat,
           LEAD(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS next_at
    FROM events e
    WHERE (e.payload->>'datacallid')::int = $%d
      AND e.resource IN ('questionnaire', 'public.scores')
      AND (e.payload->>'fismasystemid')::int IN (SELECT fismasystemid FROM scoped_systems)
),
dwell AS (
    SELECT userid,
           fismasystemid,
           questionid,
           COALESCE(readonly, FALSE) AS readonly,
           EXTRACT(EPOCH FROM LEAST(next_at - createdat, INTERVAL '%d seconds')) AS secs
    FROM stream
    WHERE action = 'viewed' AND next_at IS NOT NULL
)
SELECT d.fismasystemid,
       d.userid,
       COALESCE(u.fullname, '') AS name,
       COALESCE(u.email, '')    AS email,
       COALESCE(u.role, '')     AS role,
       d.questionid,
       SUM(d.secs) FILTER (WHERE NOT d.readonly) AS editor_seconds,
       SUM(d.secs) FILTER (WHERE d.readonly)     AS viewer_seconds
FROM dwell d
LEFT JOIN users u ON u.userid = d.userid
GROUP BY d.fismasystemid, d.userid, u.fullname, u.email, u.role, d.questionid
ORDER BY d.fismasystemid, d.userid, d.questionid
`, where, dataCallArg, idleCapSeconds)

	return sql, args
}

func scanTimeSpentRow(row pgx.CollectableRow) (*timeSpentRow, error) {
	var r timeSpentRow
	// editor/viewer come back as SUM(...) which is NULL when the FILTER matches
	// no rows; scan into nullable holders and coalesce to 0.
	var editor, viewer *float64
	if err := row.Scan(&r.FismaSystemID, &r.UserID, &r.Name, &r.Email, &r.Role, &r.QuestionID, &editor, &viewer); err != nil {
		return nil, err
	}
	if editor != nil {
		r.EditorSeconds = *editor
	}
	if viewer != nil {
		r.ViewerSeconds = *viewer
	}
	return &r, nil
}

// rollupTimeSpent aggregates the (system, user, question) grain into one
// TimeSpent per system with per-person and per-question breakdowns. Input is
// ordered by (system, user, question); the rollup does not rely on that ordering
// beyond producing a stable, grouped result.
func rollupTimeSpent(rows []*timeSpentRow) []*TimeSpent {
	// Preserve first-seen system order for a stable response.
	order := []int32{}
	systems := map[int32]*TimeSpent{}
	// Per system: distinct questions measured (denominator for the average).
	systemQuestions := map[int32]map[int32]struct{}{}
	// Per (system,user): the person accumulator and their distinct questions.
	people := map[int32]map[string]*TimeSpentPerson{}
	personQuestions := map[int32]map[string]map[int32]struct{}{}
	// Per (system,question): accumulated seconds and distinct people, plus a
	// stable first-seen order and the accumulator itself.
	questionOrder := map[int32][]int32{}
	questions := map[int32]map[int32]*TimeSpentQuestion{}
	questionSeconds := map[int32]map[int32]float64{}
	questionPeople := map[int32]map[int32]map[string]struct{}{}

	for _, r := range rows {
		secs := r.seconds()
		ts, ok := systems[r.FismaSystemID]
		if !ok {
			ts = &TimeSpent{FismaSystemID: r.FismaSystemID}
			systems[r.FismaSystemID] = ts
			order = append(order, r.FismaSystemID)
			systemQuestions[r.FismaSystemID] = map[int32]struct{}{}
			people[r.FismaSystemID] = map[string]*TimeSpentPerson{}
			personQuestions[r.FismaSystemID] = map[string]map[int32]struct{}{}
			questions[r.FismaSystemID] = map[int32]*TimeSpentQuestion{}
			questionSeconds[r.FismaSystemID] = map[int32]float64{}
			questionPeople[r.FismaSystemID] = map[int32]map[string]struct{}{}
		}

		ts.EditorSeconds += r.EditorSeconds
		ts.ViewerSeconds += r.ViewerSeconds
		ts.TotalSeconds += secs
		systemQuestions[r.FismaSystemID][r.QuestionID] = struct{}{}

		person, ok := people[r.FismaSystemID][r.UserID]
		if !ok {
			person = &TimeSpentPerson{
				AuditRef: AuditRef{UserID: r.UserID, Name: r.Name, Email: r.Email, Role: r.Role},
			}
			people[r.FismaSystemID][r.UserID] = person
			ts.PerPerson = append(ts.PerPerson, person)
			personQuestions[r.FismaSystemID][r.UserID] = map[int32]struct{}{}
		}
		person.EditorSeconds += r.EditorSeconds
		person.ViewerSeconds += r.ViewerSeconds
		person.TotalSeconds += secs
		personQuestions[r.FismaSystemID][r.UserID][r.QuestionID] = struct{}{}

		q, ok := questions[r.FismaSystemID][r.QuestionID]
		if !ok {
			q = &TimeSpentQuestion{QuestionID: r.QuestionID}
			questions[r.FismaSystemID][r.QuestionID] = q
			questionOrder[r.FismaSystemID] = append(questionOrder[r.FismaSystemID], r.QuestionID)
			questionPeople[r.FismaSystemID][r.QuestionID] = map[string]struct{}{}
		}
		questionSeconds[r.FismaSystemID][r.QuestionID] += secs
		questionPeople[r.FismaSystemID][r.QuestionID][r.UserID] = struct{}{}
	}

	result := make([]*TimeSpent, 0, len(order))
	for _, id := range order {
		ts := systems[id]
		ts.QuestionsMeasured = int32(len(systemQuestions[id]))
		if ts.QuestionsMeasured > 0 {
			ts.AverageSecondsPerQuestion = ts.TotalSeconds / float64(ts.QuestionsMeasured)
		}
		for _, p := range ts.PerPerson {
			p.QuestionsMeasured = int32(len(personQuestions[id][p.UserID]))
		}
		for _, qid := range questionOrder[id] {
			q := questions[id][qid]
			ppl := len(questionPeople[id][qid])
			q.People = int32(ppl)
			// Per-person average on this question: total time on it divided by the
			// distinct people who touched it.
			if ppl > 0 {
				q.AverageSecondsPerPerson = questionSeconds[id][qid] / float64(ppl)
			}
			ts.PerQuestion = append(ts.PerQuestion, q)
		}
		result = append(result, ts)
	}
	return result
}

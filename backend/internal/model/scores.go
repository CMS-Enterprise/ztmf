package model

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// rawQuery wraps a hand-built SQL string and its arguments so it can flow
// through the existing query helper. Used for the pillar aggregation query
// where the derived subqueries and conditional joins exceed squirrel's
// ergonomics and a parameterized string is the clearer expression.
//
// SELECT-only. The model package's queryRow helper records events derived
// from the SqlBuilder shape, so write-path callers must not use rawQuery.
type rawQuery struct {
	sql  string
	args []any
}

func (r rawQuery) ToSql() (string, []any, error) {
	return r.sql, r.args, nil
}

type Score struct {
	ScoreID          int32           `json:"scoreid"`
	FismaSystemID    int32           `json:"fismasystemid"`
	DateCalculated   float64         `json:"datecalculated"`
	Notes            *string         `json:"notes"`
	NotesIsAISummary *bool           `json:"notes_is_ai_summary" db:"notes_is_ai_summary"`
	FunctionOptionID int32           `json:"functionoptionid"`
	DataCallID       int32           `json:"datacallid"`
	FunctionOption   *FunctionOption `json:"functionoption,omitempty"`
	LastEditedAt     *time.Time      `json:"last_edited_at,omitempty"`
	LastEditedBy     *AuditRef       `json:"last_edited_by,omitempty"`
}

// AuditInfo satisfies Auditable. Returned pointers may be nil if the row has
// no recorded edit (e.g. a seed row inserted outside the event-tracking write
// path).
func (s *Score) AuditInfo() (*time.Time, *AuditRef) {
	return s.LastEditedAt, s.LastEditedBy
}

func (s *Score) Save(ctx context.Context) (*Score, error) {
	var sqlb SqlBuilder

	if err := s.validate(ctx); err != nil {
		return nil, err
	}

	// Audit-preserving no-op: on an UPDATE that does not actually change
	// any answer field, skip the write entirely. The questionnaire UI
	// PUTs on every Next click regardless of whether the user touched
	// the answer, so without this guard a read-through user gets stamped
	// as the new editor and the prior cycle's real editor is overwritten.
	// The product rule is "save on real change, not on read-through" so
	// we enforce it here as defense in depth even if the client adds its
	// own dirty check.
	//
	// Treats nil notes and empty-string notes as the same value, since the
	// FE may submit either for an unanswered notes box. Returns the
	// current row through the same lookupScoreAudit path the normal write
	// uses, so the caller cannot tell a no-op apart from a successful
	// write -- only the events table (unchanged) reveals the truth.
	//
	// On a no-op match we carry the incoming.FunctionOption (if any)
	// onto the returned current row so callers that requested
	// ?include=functionoption still get a fully-shaped response, matching
	// the populated-write path through queryRow + FindScores. The PUT
	// controller writes 204 today but still encodes the body, so the
	// response is observable to clients that parse 204 bodies.
	if s.ScoreID != 0 {
		if same, current, err := scoreUpdateIsNoOp(ctx, s); err != nil {
			return nil, err
		} else if same {
			if s.FunctionOption != nil {
				current.FunctionOption = s.FunctionOption
			}
			if at, by := lookupScoreAudit(ctx, current.ScoreID); at != nil && by != nil {
				current.LastEditedAt = at
				current.LastEditedBy = by
			}
			return current, nil
		}
	}

	if s.ScoreID == 0 {
		sqlb = stmntBuilder.
			Insert("public.scores").
			Columns("fismasystemid", "notes", "notes_is_ai_summary", "functionoptionid", "datacallid").
			Values(s.FismaSystemID, s.Notes, derefBool(s.NotesIsAISummary), s.FunctionOptionID, s.DataCallID).
			Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, notes_is_ai_summary, functionoptionid, datacallid")
	} else {
		setCols := squirrel.Eq{
			"fismasystemid":    s.FismaSystemID,
			"notes":            s.Notes,
			"functionoptionid": s.FunctionOptionID,
			"datacallid":       s.DataCallID,
		}
		if s.NotesIsAISummary != nil {
			setCols["notes_is_ai_summary"] = *s.NotesIsAISummary
		}
		sqlb = stmntBuilder.
			Update("public.scores").
			SetMap(setCols).
			Where("scoreid=?", s.ScoreID).
			Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, notes_is_ai_summary, functionoptionid, datacallid")
	}

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[Score])
	if err != nil {
		return saved, err
	}

	// Stamp the just-performed edit onto the response so the POST/PUT body
	// is consistent with what a subsequent GET will return. We read back
	// the canonical row that recordEvent (fired from queryRow above) just
	// wrote, rather than synthesizing from time.Now() + ctx user. Two
	// reasons:
	//   1) recordEvent currently logs-and-swallows errors. If the event
	//      INSERT failed (FK, JSONB issue, transient), no event row
	//      exists. Reading back means we leave audit fields nil instead
	//      of advertising a phantom editor the next GET cannot confirm.
	//   2) Postgres CURRENT_TIMESTAMP is the authoritative source. Using
	//      time.Now().UTC() invites sub-second clock skew between the
	//      stamped response and the canonical events.createdat that
	//      subsequent reads project. Same source = no drift.
	if saved != nil {
		// Both-or-neither: only stamp when the lateral lookup resolved
		// both the event timestamp AND the editor identity. See the
		// matching scan logic in FindScores below for the same rule.
		if at, by := lookupScoreAudit(ctx, saved.ScoreID); at != nil && by != nil {
			saved.LastEditedAt = at
			saved.LastEditedBy = by
		}
	}
	return saved, nil
}

// scoreUpdateIsNoOp reports whether the incoming Score's answer fields
// match the existing row exactly. Used by Save to short-circuit the
// "Next click without editing" path so the prior editor is not
// overwritten by a read-through. Returns the existing row alongside the
// boolean so callers can return it as the response without a second
// round trip.
//
// notes is compared as a string with nil normalized to "" -- the FE may
// submit either for an unanswered notes box and we treat them as the
// same value. Whitespace-only notes intentionally do NOT normalize to
// empty; the FE trims before submitting today, so reaching this layer
// with " " means a caller is sending whitespace deliberately and the
// stored value should reflect that. See TestScoresEqualForUpdate's
// WhitespaceNotesNotEqualEmpty case for the pinned contract.
// fismasystemid and datacallid are included in the comparison because
// a PUT that moves a score across systems or cycles is a real change
// even if notes and option are unchanged.
//
// Concurrency: this SELECT precedes the (skipped) UPDATE outside any
// transaction, so a concurrent writer could land a real change in the
// gap. The window is small and the practical consequence is "the next
// GET corrects the audit fields" -- there is no data loss, only a
// brief response that lags the latest write. Wrapping Save in a
// transaction would close the window but rework every model that
// relies on queryRow's auto-event hook, which is out of scope for the
// audit-fields branch.
//
// Returns ErrNotData when the row is missing so the caller can fail
// cleanly without paying a second round trip through the UPDATE that
// would also fail.
func scoreUpdateIsNoOp(ctx context.Context, incoming *Score) (bool, *Score, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, nil, trapError(err)
	}
	defer conn.Release()

	current := &Score{}
	err = conn.QueryRow(ctx, `
		SELECT scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) AS datecalculated,
		       notes, notes_is_ai_summary, functionoptionid, datacallid
		FROM scores WHERE scoreid = $1
	`, incoming.ScoreID).Scan(
		&current.ScoreID, &current.FismaSystemID, &current.DateCalculated,
		&current.Notes, &current.NotesIsAISummary, &current.FunctionOptionID, &current.DataCallID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil, ErrNoData
		}
		return false, nil, trapError(err)
	}

	return scoresEqualForUpdate(current, incoming), current, nil
}

// scoresEqualForUpdate is the pure comparison used by scoreUpdateIsNoOp,
// extracted so unit tests can pin the equality rules without spinning up
// a database. Returns true when the incoming Score would produce no
// observable change to the answer fields on the current row.
func scoresEqualForUpdate(current, incoming *Score) bool {
	if current == nil || incoming == nil {
		return false
	}
	if current.FismaSystemID != incoming.FismaSystemID ||
		current.DataCallID != incoming.DataCallID ||
		current.FunctionOptionID != incoming.FunctionOptionID {
		return false
	}
	if incoming.NotesIsAISummary != nil && derefBool(current.NotesIsAISummary) != *incoming.NotesIsAISummary {
		return false
	}
	return derefString(current.Notes) == derefString(incoming.Notes)
}

// lookupScoreAudit fetches the most recent event for the given scoreid
// and resolves the editor identity from users. Returns (nil, nil) when
// no event row exists -- the caller MUST treat that as the "do not
// stamp" signal rather than substituting synthetic values, otherwise
// the POST/PUT response will diverge from what subsequent FindScores
// reads return through the same lateral join (see [[scores.FindScores]]).
//
// Resource literal 'public.scores' mirrors the legacy writer in Save
// above; bare-name normalization is tracked as a separate follow-up.
func lookupScoreAudit(ctx context.Context, scoreID int32) (*time.Time, *AuditRef) {
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Println("lookupScoreAudit: db.Conn:", err)
		return nil, nil
	}
	defer conn.Release()

	const q = `
		SELECT e.createdat, u.userid, u.fullname, u.email, u.role
		FROM events e
		LEFT JOIN users u ON u.userid = e.userid
		WHERE e.resource = 'public.scores'
		  AND (e.payload->>'scoreid')::int = $1
		ORDER BY e.createdat DESC
		LIMIT 1
	`
	var (
		at    time.Time
		uid   *string
		name  *string
		email *string
		role  *string
	)
	if err := conn.QueryRow(ctx, q, scoreID).Scan(&at, &uid, &name, &email, &role); err != nil {
		// No row is the expected outcome when recordEvent skipped/failed.
		// Anything else is genuinely unexpected; log it for forensics but
		// stay on the "do not stamp" path so the response cannot lie.
		return nil, nil
	}
	if uid == nil {
		// Event exists but editor was nil at write time (no user in ctx).
		// Surface the timestamp; leave LastEditedBy nil so the caller can
		// decide whether to clear both (see Save).
		return &at, nil
	}
	return &at, &AuditRef{
		UserID: *uid,
		Name:   derefString(name),
		Email:  derefString(email),
		Role:   derefString(role),
	}
}

func (s *Score) validate(ctx context.Context) error {

	if s.Notes != nil && utf8.RuneCountInString(*s.Notes) > 2000 {
		return ErrNotesTooLong
	}

	dataCall, err := FindDataCallByID(ctx, s.DataCallID)
	if err != nil {
		return err
	}

	user := UserFromContext(ctx)
	if time.Now().UTC().After(dataCall.Deadline) && (user == nil || !user.IsAdmin()) {
		return ErrPastDeadline
	}

	return nil
}

type ScoreAggregate struct {
	DataCallID    int32          `json:"datacallid"`
	FismaSystemID int32          `json:"fismasystemid"`
	SystemScore   float64        `json:"systemscore"`
	SystemTier    string         `json:"systemtier"`
	PillarScores  []*PillarScore `json:"pillarscores,omitempty" db:"-"`
}

type PillarScore struct {
	PillarID int32   `json:"pillarid"`
	Pillar   string  `json:"pillar"`
	Score    float64 `json:"score"`
	Tier     string  `json:"tier"`
}

// Tier returns the HHS-aligned maturity tier label for a 1.0-5.0 score.
// Boundaries (on the score rounded to two decimal places, matching the
// frontend display):
//
//	>= 4.10  -> Optimal
//	>= 3.10  -> Advanced
//	>= 2.10  -> Initial
//	>= 1.01  -> Traditional
//	otherwise -> Not Assessed (a pillar with zero answered questions lands
//	             at exactly 1.0 under the +1 shift aggregation).
//
// The comparison happens in integer space (score * 100 rounded to int)
// so a system whose float64 representation is, e.g., 3.099999... but
// displays to the user as "3.10" via toFixed(2) is correctly classified
// the same way the user sees it. IEEE 754 cannot represent 4.1, 3.1, or
// 2.1 exactly, so a direct `score >= 4.1` comparison would mis-tier
// inputs that are arithmetically at the boundary but stored a few ulps
// low. Integer comparison removes that ambiguity entirely.
func Tier(score float64) string {
	hundredths := int(math.Round(score * 100))
	switch {
	case hundredths >= 410:
		return "Optimal"
	case hundredths >= 310:
		return "Advanced"
	case hundredths >= 210:
		return "Initial"
	case hundredths >= 101:
		return "Traditional"
	default:
		return "Not Assessed"
	}
}

type FindScoresInput struct {
	input
	FismaSystemID  *int32 `schema:"fismasystemid"`
	FismaSystemIDs []*int32
	DataCallID     *int32 `schema:"datacallid"`
	UserID         *string
	IncludePillars *bool `schema:"include_pillars"`
	// OpDiv scope for OpDiv-scoped admin tiers. Restricts scores to systems in
	// the admin's granted OpDivs; RestrictToOpDivIDs with an empty slice fails
	// closed. Not schema-tagged - the controller sets them from the auth'd user.
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

func FindScores(ctx context.Context, input FindScoresInput) ([]*Score, error) {

	sqlb := stmntBuilder.
		Select("scoreid, scores.fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, scores.notes_is_ai_summary, scores.functionoptionid, scores.datacallid").
		From("scores")

	if input.contains("functionoption") {
		sqlb = sqlb.
			Columns(functionOptionColumns...).
			InnerJoin("functionoptions on functionoptions.functionoptionid=scores.functionoptionid")
	}

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid=scores.fismasystemid AND users_fismasystems.userid=?", *input.UserID)
	}

	if input.FismaSystemID != nil {
		sqlb = sqlb.Where("scores.fismasystemid=?", *input.FismaSystemID)
	}

	if input.DataCallID != nil {
		sqlb = sqlb.Where("datacallid=?", *input.DataCallID)
	}

	// OpDiv scope (fail-closed): restrict to scores whose system belongs to one
	// of the admin's granted OpDivs. Empty grants under RestrictToOpDivIDs ->
	// no rows. Expressed as a subquery predicate so it composes with the audit
	// joins below without disturbing their arg ordering.
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		sqlb = sqlb.Where("FALSE")
	case len(input.OpDivIDs) > 0:
		sqlb = sqlb.Where("scores.fismasystemid IN (SELECT fismasystemid FROM fismasystems WHERE opdiv_id = ANY(?))", input.OpDivIDs)
	}

	// Attach last-edit audit info. The lateral subquery picks the most recent
	// write event for the row; the outer left join resolves the editor's
	// identity. Both joins are LEFT so a score row missing an event (seed
	// data) still returns. The 'public.scores' literal matches the value
	// recordEvent writes from scores.Save; see follow-up to normalize.
	sqlb = sqlb.
		JoinClause(`LEFT JOIN LATERAL (
			SELECT createdat, userid
			FROM events
			WHERE resource = 'public.scores'
			  AND (payload->>'scoreid')::int = scores.scoreid
			ORDER BY createdat DESC
			LIMIT 1
		) last_edit ON TRUE`).
		LeftJoin("users last_edited_user ON last_edited_user.userid = last_edit.userid").
		Columns(
			"last_edit.createdat AS last_edited_at",
			"last_edited_user.userid AS last_edited_by_userid",
			"last_edited_user.fullname AS last_edited_by_name",
			"last_edited_user.email AS last_edited_by_email",
			"last_edited_user.role AS last_edited_by_role",
		)

	return query(ctx, sqlb, func(row pgx.CollectableRow) (*Score, error) {
		score := Score{}
		fields := []any{&score.ScoreID, &score.FismaSystemID, &score.DateCalculated, &score.Notes, &score.NotesIsAISummary, &score.FunctionOptionID, &score.DataCallID}
		if input.contains("functionoption") {
			score.FunctionOption = &FunctionOption{}
			fields = append(fields, &score.FunctionOption.FunctionOptionID, &score.FunctionOption.FunctionID, &score.FunctionOption.Score, &score.FunctionOption.OptionName, &score.FunctionOption.Description)
		}
		var (
			lastEditedAt *time.Time
			editorUserID *string
			editorName   *string
			editorEmail  *string
			editorRole   *string
		)
		fields = append(fields, &lastEditedAt, &editorUserID, &editorName, &editorEmail, &editorRole)
		if err := row.Scan(fields...); err != nil {
			return &score, err
		}
		// Both-or-neither: a populated audit pair means both fields
		// resolved cleanly from the lateral join. If either side is
		// missing (no event row for this scoreid, or an event row whose
		// editor userid no longer resolves through the users join), we
		// emit nothing rather than half a record. Lets the frontend
		// treat "absent" as a single state instead of branching on each
		// field independently. Encoded in OpenAPI as both nullable +
		// omitempty.
		if lastEditedAt != nil && editorUserID != nil {
			score.LastEditedAt = lastEditedAt
			score.LastEditedBy = &AuditRef{
				UserID: *editorUserID,
				Name:   derefString(editorName),
				Email:  derefString(editorEmail),
				Role:   derefString(editorRole),
			}
		}
		return &score, nil
	})
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// pillarScoreRow is the wire shape returned by findPillarScoresAll. One row
// per (datacall, system, pillar). Both pillar Score and the carry-along
// SystemScore are computed in Postgres and emerge on the HHS 1.0-5.0 scale,
// so the Go aggregation step never re-computes them and float precision is
// consistent across boundaries.
//
// Field tagging: pgx.RowToAddrOfStructByName reads the `db` struct tag (and
// falls back to case-insensitive name match), not the `json` tag. The wire
// names here intentionally match the SQL select aliases via `db`; do not
// switch to `json` tags by reflex when refactoring.
type pillarScoreRow struct {
	DataCallID    int32   `db:"datacallid"`
	FismaSystemID int32   `db:"fismasystemid"`
	PillarID      int32   `db:"pillarid"`
	Pillar        string  `db:"pillar"`
	Score         float64 `db:"score"`
	SystemScore   float64 `db:"system_score"`
}

// FindScoresAggregate returns one ScoreAggregate per (datacall, system) pair
// matching the input filters. Both the system score and the per-pillar scores
// are computed on the HHS-aligned 1.0-5.0 scale via the +1 shift aggregation
// described in ztmf-misc#175. The system score is the simple AVG of pillar
// scores; the divisor follows the actual pillar count rather than being
// hardcoded, so adding or removing a pillar does not silently change the
// math. Pillar scores are populated on the aggregate when IncludePillars is
// true.
func FindScoresAggregate(ctx context.Context, input FindScoresInput) ([]*ScoreAggregate, error) {
	// Convert single FismaSystemID to FismaSystemIDs array if needed so the
	// scoping rules below see a consistent shape. This mirrors prior behavior
	// so an admin requesting a specific system still flows through the same
	// IN-list path that ISSOs use for their assigned-system scope.
	if input.FismaSystemID != nil && len(input.FismaSystemIDs) == 0 {
		input.FismaSystemIDs = []*int32{input.FismaSystemID}
	}

	pillarRows, err := findPillarScoresAll(ctx, input)
	if err != nil {
		return nil, err
	}

	includePillars := input.IncludePillars != nil && *input.IncludePillars
	return aggregatePillarRows(pillarRows, includePillars), nil
}

// aggregatePillarRows is the pure-Go rollup step extracted so unit tests can
// pin the response shape without spinning up a database. It collapses the
// per-(datacall, system, pillar) rows produced by findPillarScoresAll into
// one ScoreAggregate per (datacall, system) pair, attaches the Tier label
// derived from each pillar score, and reads the carry-along SystemScore that
// the underlying SQL already averaged via a window function.
//
// All score math happens in Postgres. This function does no arithmetic on
// pillar scores beyond reading the system_score field on the first row of
// each (datacall, system) group; that keeps float precision consistent
// across the SQL/Go boundary and removes any chance of a tier flip from
// recomputing the same average in a different runtime.
//
// The input slice is assumed to be ordered by (datacallid, fismasystemid,
// pillarid), which is the canonical ordering emitted by the underlying SQL
// in findPillarScoresAll. The output preserves that order.
func aggregatePillarRows(rows []*pillarScoreRow, includePillars bool) []*ScoreAggregate {
	type key struct {
		dataCallID    int32
		fismaSystemID int32
	}

	aggByKey := map[key]*ScoreAggregate{}
	order := []key{}

	for _, r := range rows {
		k := key{r.DataCallID, r.FismaSystemID}
		agg, seen := aggByKey[k]
		if !seen {
			agg = &ScoreAggregate{
				DataCallID:    r.DataCallID,
				FismaSystemID: r.FismaSystemID,
				SystemScore:   r.SystemScore,
				SystemTier:    Tier(r.SystemScore),
			}
			aggByKey[k] = agg
			order = append(order, k)
		}
		if includePillars {
			agg.PillarScores = append(agg.PillarScores, &PillarScore{
				PillarID: r.PillarID,
				Pillar:   r.Pillar,
				Score:    r.Score,
				Tier:     Tier(r.Score),
			})
		}
	}

	aggregates := make([]*ScoreAggregate, 0, len(order))
	for _, k := range order {
		aggregates = append(aggregates, aggByKey[k])
	}
	return aggregates
}

// findPillarScoresAll returns one row per (datacall, system, pillar) for every
// combination matching the input filters. Pillar scores are computed by
// enumerating every expected question for each (system, datacall, pillar)
// triple, LEFT JOINing to existing answers, COALESCEing missing rows to 0,
// applying the +1 shift, and averaging.
//
// The query is built as parameterized raw SQL because the derived subqueries
// and conditional joins exceed what squirrel expresses cleanly. Filters are
// pushed into the `expected` subquery so the LEFT JOIN only operates on the
// scoped set.
func findPillarScoresAll(ctx context.Context, input FindScoresInput) ([]*pillarScoreRow, error) {
	sql, args := buildPillarScoresSQL(input)
	return query(ctx, rawQuery{sql: sql, args: args}, pgx.RowToAddrOfStructByName[pillarScoreRow])
}

// buildPillarScoresSQL assembles the parameterized SQL for the pillar
// aggregation. Extracted so unit tests can verify the filter and scope
// shaping without a database connection.
func buildPillarScoresSQL(input FindScoresInput) (string, []any) {
	var conds []string
	var args []any
	argN := 1

	// No implicit decommissioned filter. The legacy aggregate query had none,
	// so historical scoring for systems that have since been decommissioned
	// stayed reachable through this endpoint. Re-applying that filter here
	// would regress audit / history use cases. Callers that want only active
	// systems should filter at the controller layer.
	conds = append(conds, "TRUE")

	if input.DataCallID != nil {
		conds = append(conds, fmt.Sprintf("dc.datacallid = $%d", argN))
		args = append(args, *input.DataCallID)
		argN++
	}

	if input.FismaSystemID != nil {
		conds = append(conds, fmt.Sprintf("fs.fismasystemid = $%d", argN))
		args = append(args, *input.FismaSystemID)
		argN++
	}

	if len(input.FismaSystemIDs) > 0 {
		placeholders := make([]string, len(input.FismaSystemIDs))
		for i, id := range input.FismaSystemIDs {
			placeholders[i] = fmt.Sprintf("$%d", argN)
			args = append(args, id)
			argN++
		}
		conds = append(conds, fmt.Sprintf("fs.fismasystemid IN (%s)", strings.Join(placeholders, ",")))
	}

	// OpDiv scope (fail-closed) on the expected CTE, which already joins
	// fismasystems as fs. Empty grants under RestrictToOpDivIDs -> FALSE.
	switch {
	case input.RestrictToOpDivIDs && len(input.OpDivIDs) == 0:
		conds = append(conds, "FALSE")
	case len(input.OpDivIDs) > 0:
		conds = append(conds, fmt.Sprintf("fs.opdiv_id = ANY($%d)", argN))
		args = append(args, input.OpDivIDs)
		argN++
	}

	var userJoin string
	if input.UserID != nil {
		userJoin = fmt.Sprintf("INNER JOIN users_fismasystems ufs ON ufs.fismasystemid = fs.fismasystemid AND ufs.userid = $%d", argN)
		args = append(args, *input.UserID)
		argN++
	}

	// The (system, datacall) universe is the set of pairs that appear at
	// least once in the scores table. This preserves the legacy aggregate
	// contract: a system that was never scored for a given data call did
	// not appear in /scores/aggregate before this change, and still does
	// not after. The frontend already cross-references /fismasystems to
	// render "No Score" tiles for systems that have not started, so a
	// blanket cartesian over every active system would double-count.
	//
	// Pillar-level "Not Assessed" is still reachable for partial systems:
	// once a (system, datacall) pair enters the universe via any score,
	// every pillar for that system gets enumerated through the expected
	// CTE. Pillars where every function is unanswered COALESCE to 0,
	// shift to 1.0, average to exactly 1.0, and emit Not Assessed.
	//
	// The datacalls_fismasystems junction looks like the natural fit here
	// but is misleading: per backend/internal/model/datacallsfismasystems.go,
	// rows are written only when an ISSO marks a data call as complete,
	// not when systems are enrolled. Production data confirms this — the
	// junction is empty for live data calls that have thousands of score
	// rows. Using it as the universe filter returns nothing.
	//
	// Question catalog drift is intentionally not handled. Functions are
	// keyed by datacenterenvironment, not by datacall. A retroactive
	// recompute of a closed historical cycle uses the current question set
	// for that environment, which matches the legacy AVG-over-scored-rows
	// behavior (it only averaged whatever functionoptions were referenced
	// in the scores table). If question versioning becomes a real
	// requirement, both paths would need the same treatment.
	//
	// Both pillar score and system score are computed in Postgres so the
	// float math is consistent. System score is AVG of pillar scores
	// (equal weighting per the locked plan); the divisor follows the
	// actual pillar count via the inner AVG, so adding or removing a
	// pillar does not silently shift the math. system_score is carried on
	// every pillar row via a window function so callers that want only
	// the system roll-up read it from any row without a second query.
	sql := fmt.Sprintf(`
WITH scored_pairs AS (
    SELECT DISTINCT fismasystemid, datacallid FROM scores
),
expected AS (
    SELECT sp.fismasystemid, sp.datacallid, p.pillarid, p.pillar, f.functionid
    FROM scored_pairs sp
    INNER JOIN fismasystems fs ON fs.fismasystemid = sp.fismasystemid
    INNER JOIN datacalls dc    ON dc.datacallid    = sp.datacallid
    -- Resolve the system's raw datacenterenvironment to its scoring vocabulary
    -- via the mapping table (ztmf#392), then match functions on that key. A raw
    -- value with no mapping row, or one whose scoring_key is NULL (e.g. the
    -- DECOMMISSIONED marker), matches no functions and is excluded from scoring,
    -- exactly as an unrecognized value was under the old direct join.
    INNER JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
    INNER JOIN functions f ON f.datacenterenvironment = dce.scoring_key
    INNER JOIN questions q ON q.questionid = f.questionid
    INNER JOIN pillars p   ON p.pillarid   = q.pillarid
    %s
    WHERE %s
),
answers AS (
    SELECT s.fismasystemid, s.datacallid, fo.functionid, fo.score
    FROM scores s
    INNER JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
),
pillar_scores AS (
    SELECT
        e.datacallid,
        e.fismasystemid,
        e.pillarid,
        e.pillar,
        AVG(COALESCE(a.score, 0) + 1.0)::float8 AS pillar_score
    FROM expected e
    LEFT JOIN answers a
      ON a.fismasystemid = e.fismasystemid
     AND a.datacallid    = e.datacallid
     AND a.functionid    = e.functionid
    GROUP BY e.datacallid, e.fismasystemid, e.pillarid, e.pillar
)
SELECT
    ps.datacallid,
    ps.fismasystemid,
    ps.pillarid,
    ps.pillar,
    ps.pillar_score AS score,
    AVG(ps.pillar_score) OVER (PARTITION BY ps.datacallid, ps.fismasystemid)::float8 AS system_score
FROM pillar_scores ps
ORDER BY ps.datacallid, ps.fismasystemid, ps.pillarid
`, userJoin, strings.Join(conds, " AND "))

	return sql, args
}

// copyPreviousScores rolls the previous cycle's answers forward into the
// newly created data call identified by dataCallID (the *latest* datacall; the
// previous one is discovered via findPreviousDataCall). It returns the number
// of score rows copied.
//
// The rollover is best-effort enrichment - it never invalidates the new data
// call - but a zero, partial, or errored copy is surfaced loudly via the
// ROLLOVER_ANOMALY log token (wired to a CloudWatch metric alarm) so an empty
// cycle is detected rather than shipped silently. See ztmf#411.
func copyPreviousScores(ctx context.Context, dataCallID int32) (int64, error) {
	prevDataCall, err := findPreviousDataCall(dataCallID)
	if err != nil {
		// No previous cycle (the first-ever data call) is the expected, benign
		// case: nothing to roll forward, and not an anomaly.
		if errors.Is(err, ErrNoData) {
			return 0, nil
		}
		log.Printf("ROLLOVER_ANOMALY datacall=%d expected=? copied=0 err=%v", dataCallID, err)
		return 0, err
	}

	// skip convenience methods to avoid recording events for this operation
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Printf("ROLLOVER_ANOMALY datacall=%d expected=? copied=0 err=%v", dataCallID, err)
		return 0, err
	}
	defer conn.Release()

	// Total candidate rows in the previous cycle, including any that are no
	// longer referentially valid. Compared against the count actually copied to
	// detect a partial (or empty) rollover.
	var expected int64
	countSql, countArgs, _ := squirrel.
		Select("COUNT(*)").
		From("scores").
		Where("datacallid=?", prevDataCall.DataCallID).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err = conn.QueryRow(ctx, countSql, countArgs...).Scan(&expected); err != nil {
		log.Printf("ROLLOVER_ANOMALY datacall=%d expected=? copied=0 err=%v", dataCallID, err)
		return 0, err
	}

	// Copy the previous scores into the new cycle, rewriting datacallid to the
	// latest. INNER JOIN the FK parents so a score whose fismasystem or
	// functionoption no longer resolves is silently dropped instead of aborting
	// the entire batch - the all-or-nothing INSERT...SELECT was the ztmf#411
	// foot-gun where a single bad row emptied the whole rollover.
	prevScoresSqlb := squirrel.
		Select("s.fismasystemid", "s.datecalculated", "s.notes", "s.notes_is_ai_summary", "s.functionoptionid", fmt.Sprintf("%d as latestdatacallid", dataCallID)).
		From("scores s").
		Join("fismasystems fs ON fs.fismasystemid = s.fismasystemid").
		Join("functionoptions fo ON fo.functionoptionid = s.functionoptionid").
		Where("s.datacallid=?", prevDataCall.DataCallID)

	sqlb := squirrel.
		Insert("scores").
		Columns("fismasystemid", "datecalculated", "notes", "notes_is_ai_summary", "functionoptionid", "datacallid").
		Select(prevScoresSqlb).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, _ := sqlb.ToSql()

	tag, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		log.Printf("ROLLOVER_ANOMALY datacall=%d expected=%d copied=0 err=%v", dataCallID, expected, err)
		return 0, err
	}

	copied := tag.RowsAffected()
	if copied < expected {
		// Either zero-when-expected, or a partial copy where dead-FK rows were
		// dropped by the JOIN. Both warrant an operator signal.
		log.Printf("ROLLOVER_ANOMALY datacall=%d expected=%d copied=%d err=<none>", dataCallID, expected, copied)
	}

	return copied, nil
}

package model

import (
	"context"
	"fmt"
	"log"
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
	FunctionOptionID int32           `json:"functionoptionid"`
	DataCallID       int32           `json:"datacallid"`
	FunctionOption   *FunctionOption `json:"functionoption,omitempty"`
}

func (s *Score) Save(ctx context.Context) (*Score, error) {
	var sqlb SqlBuilder

	if err := s.validate(ctx); err != nil {
		return nil, err
	}

	if s.ScoreID == 0 {
		sqlb = stmntBuilder.
			Insert("public.scores").
			Columns("fismasystemid", "notes", "functionoptionid", "datacallid").
			Values(s.FismaSystemID, s.Notes, s.FunctionOptionID, s.DataCallID).
			Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid")
	} else {
		sqlb = stmntBuilder.
			Update("public.scores").
			Set("fismasystemid", s.FismaSystemID).
			Set("notes", s.Notes).
			Set("functionoptionid", s.FunctionOptionID).
			Set("datacallid", s.DataCallID).
			Where("scoreid=?", s.ScoreID).
			Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid")
	}
	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[Score])
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
// Boundaries:
//
//	>= 4.1   -> Optimal
//	>= 3.1   -> Advanced
//	>= 2.1   -> Initial
//	>= 1.01  -> Traditional
//	otherwise -> Not Assessed (a pillar with zero answered questions lands
//	             at exactly 1.0 under the +1 shift aggregation, so the
//	             Traditional floor uses 1.01 rather than == 1.0 to keep
//	             the predicate safe under float drift).
func Tier(score float64) string {
	switch {
	case score >= 4.1:
		return "Optimal"
	case score >= 3.1:
		return "Advanced"
	case score >= 2.1:
		return "Initial"
	case score >= 1.01:
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
}

func FindScores(ctx context.Context, input FindScoresInput) ([]*Score, error) {

	sqlb := stmntBuilder.
		Select("scoreid, scores.fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, scores.functionoptionid, scores.datacallid").
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

	return query(ctx, sqlb, func(row pgx.CollectableRow) (*Score, error) {
		score := Score{}
		fields := []any{&score.ScoreID, &score.FismaSystemID, &score.DateCalculated, &score.Notes, &score.FunctionOptionID, &score.DataCallID}
		if input.contains("functionoption") {
			score.FunctionOption = &FunctionOption{}
			fields = append(fields, &score.FunctionOption.FunctionOptionID, &score.FunctionOption.FunctionID, &score.FunctionOption.Score, &score.FunctionOption.OptionName, &score.FunctionOption.Description)
		}
		err := row.Scan(fields...)
		return &score, err
	})
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
    INNER JOIN functions f ON f.datacenterenvironment = fs.datacenterenvironment
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

// dataCallID is meant to be passed the *latest* datacall most recently created so the previous can be selected
func copyPreviousScores(dataCallID int32) {
	prevDataCall, err := findPreviousDataCall(dataCallID)

	if err != nil {
		log.Println(err)
		return
	}

	// select the previous scores but set the datacallid to be the latest
	prevScoresSqlb := squirrel.
		Select("fismasystemid", "datecalculated", "notes", "functionoptionid", fmt.Sprintf("%d as latestdatacallid", dataCallID)).
		From("scores").
		Where("datacallid=?", prevDataCall.DataCallID)

	sqlb := squirrel.
		Insert("scores").
		Columns("fismasystemid", "datecalculated", "notes", "functionoptionid", "datacallid").
		Select(prevScoresSqlb).
		PlaceholderFormat(squirrel.Dollar)

	// skip convenience methods to avoid recording events for this operation
	conn, err := db.Conn(context.TODO())
	if err != nil {
		return
	}

	sql, args, _ := sqlb.ToSql()

	_, err = conn.Exec(context.TODO(), sql, args...)

	if err != nil {
		log.Println(err)
	}
}

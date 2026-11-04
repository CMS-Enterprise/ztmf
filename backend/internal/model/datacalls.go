package model

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var dataCallColumns = []string{"datacallid", "datacall", "datecreated", "deadline", "fiscal_year", "cadence"}

// Cadence values a data call can carry. They mirror the datacalls_cadence_check
// constraint.
const (
	CadenceAnnual    = "annual"
	CadenceQuarterly = "quarterly"
)

// The fiscal year bounds mirror the datacalls_fiscal_year_check constraint. The
// floor is 2000 because a two-digit "FY##" name expands to 20##. The bounds only
// catch a typo (a two-digit year, a dropped or extra digit, zero); they are not
// a policy on which years are allowed.
const (
	minFiscalYear = 2000
	maxFiscalYear = 2999
)

// An update is partial: a field left out of the body (an empty title or
// cadence, a zero deadline or fiscal year) keeps its stored value.
type DataCall struct {
	DataCallID int32 `json:"datacallid"`
	// Free-text title, unique ignoring case. Nothing parses it; the fiscal year
	// and cadence are separate fields. Required on create; omitted on update
	// keeps the stored title.
	DataCall    string    `json:"datacall"`
	DateCreated time.Time `json:"datecreated"`
	// Required on create; omitted on update keeps the stored deadline.
	Deadline time.Time `json:"deadline"`
	// Four-digit fiscal year. Required on create; omitted on update keeps the
	// stored year.
	FiscalYear int32 `json:"fiscal_year" minimum:"2000" maximum:"2999" example:"2026"`
	// Defaults to annual on create; omitted on update keeps the stored cadence.
	Cadence string `json:"cadence" enums:"annual,quarterly" example:"annual"`
}

func (d *DataCall) Save(ctx context.Context) (*DataCall, error) {

	var sqlb SqlBuilder

	if d.DataCallID == 0 && d.Cadence == "" {
		d.Cadence = CadenceAnnual
	}

	if err := d.validate(); err != nil {
		return nil, err
	}

	if d.DataCallID == 0 {
		sqlb = stmntBuilder.
			Insert("datacalls").
			Columns("datacall", "deadline", "fiscal_year", "cadence").
			Values(d.DataCall, d.Deadline, d.FiscalYear, d.Cadence).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", "))
	} else {
		// Every field keeps its stored value when the body leaves it out, so
		// PUT {"cadence":"quarterly"} changes the cadence and nothing else.
		var deadline any
		if !d.Deadline.IsZero() {
			deadline = d.Deadline
		}
		sqlb = stmntBuilder.
			Update("datacalls").
			Set("datacall", squirrel.Expr("COALESCE(NULLIF(?::text, ''), datacall)", d.DataCall)).
			Set("deadline", squirrel.Expr("COALESCE(?::timestamptz, deadline)", deadline)).
			Set("fiscal_year", squirrel.Expr("COALESCE(NULLIF(?::int, 0), fiscal_year)", d.FiscalYear)).
			Set("cadence", squirrel.Expr("COALESCE(NULLIF(?::text, ''), cadence)", d.Cadence)).
			Where("datacallid=?", d.DataCallID).
			Suffix("RETURNING " + strings.Join(dataCallColumns, ", "))
	}

	dataCall, err := queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
	if err != nil {
		return nil, err
	}

	// Roll the previous cycle's answers into a *newly created* data call only.
	// Never on update: re-running the copy on an edit would duplicate every
	// carried-over score (ztmf#411). Run it synchronously so the outcome is
	// observable, but do not fail the create on a copy error - the datacall row
	// is already committed and is valid without a rollover (the first-ever cycle
	// legitimately copies zero rows). copyPreviousScores emits the loud
	// ROLLOVER_ANOMALY signal on any zero/partial/errored copy.
	if d.DataCallID == 0 {
		if _, err := copyPreviousScores(ctx, dataCall.DataCallID); err != nil {
			log.Println(err)
		}
	}

	return dataCall, nil
}

// validate checks the fields a client sends. On update an omitted field (zero
// value) means "keep the stored value", so only fields that are present are
// checked. On create the title, deadline and fiscal year are required.
//
// An empty cadence passes on both paths. Save defaults it to annual before an
// insert, and the column's DEFAULT 'annual' covers a create that reaches the
// database without going through Save.
func (d *DataCall) validate() error {
	err := InvalidInputError{data: map[string]any{}}
	create := d.DataCallID == 0

	// TrimSpace for blankness: "   " is not a title. Only the empty string
	// means "omitted" on update.
	if d.DataCall != "" && strings.TrimSpace(d.DataCall) == "" {
		err.data["datacall"] = d.DataCall
	} else if create && d.DataCall == "" {
		err.data["datacall"] = d.DataCall
	}

	if create && d.Deadline.IsZero() {
		err.data["deadline"] = d.Deadline
	}

	if d.FiscalYear == 0 {
		if create {
			err.data["fiscal_year"] = d.FiscalYear
		}
	} else if d.FiscalYear < minFiscalYear || d.FiscalYear > maxFiscalYear {
		err.data["fiscal_year"] = d.FiscalYear
	}

	switch d.Cadence {
	case "", CadenceAnnual, CadenceQuarterly:
	default:
		err.data["cadence"] = d.Cadence
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

func FindDataCalls(ctx context.Context) ([]*DataCall, error) {
	sqlb := stmntBuilder.Select(dataCallColumns...).
		From("datacalls").
		OrderBy("deadline DESC", "datacallid DESC")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCall])
}

func FindDataCallByID(ctx context.Context, dataCallID int32) (*DataCall, error) {
	sqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("datacallid=?", dataCallID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
}

func findPreviousDataCall(ctx context.Context, dataCallID int32) (*DataCall, error) {
	// find the *previous* datacall: the most recent cycle whose deadline is
	// strictly earlier than this call's deadline. Ordering is deadline-driven
	// (not datacallid) because historical loads can carry a higher datacallid
	// than the real prior call.
	//
	// The "strictly earlier than this call's deadline" restriction (not merely
	// "the globally latest other call") is what fixes ztmf#448: a backfill data
	// call can be created with a deadline BEFORE existing cycles, and picking the
	// globally-latest other call would then resolve a future cycle as its
	// "previous" and roll that future cycle's answers backward. Anchoring to
	// deadlines before this call's makes a normal new cycle pick the real prior
	// cycle, and a backfill pick the correct earlier cycle - or none, which is
	// benign (nothing to roll forward). This also excludes the call itself, since
	// its own deadline is not strictly before itself. Among the strictly-earlier
	// candidates, datacallid DESC breaks deadline ties; a cycle sharing this
	// call's exact deadline is not a candidate at all.
	prevDcSqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		Where("deadline < (SELECT deadline FROM datacalls WHERE datacallid=?)", dataCallID).
		OrderBy("deadline DESC", "datacallid DESC").
		Limit(1)

	return queryRow(ctx, prevDcSqlb, pgx.RowToStructByName[DataCall])
}

func FindLatestDataCall(ctx context.Context) (*DataCall, error) {
	// The current/latest datacall is the one with the furthest-out deadline
	// (datacallid DESC only as a tiebreak): the annual cadence is deadline-driven,
	// and historical loads can carry a higher datacallid than the real current call.
	sqlb := stmntBuilder.
		Select(dataCallColumns...).
		From("datacalls").
		OrderBy("deadline DESC", "datacallid DESC").
		Limit(1)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[DataCall])
}

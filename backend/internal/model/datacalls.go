package model

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	dataCallColumns    = []string{"datacallid", "datacall", "datecreated", "deadline", "questionnaire_versionid"}
	dataCallColumnList = strings.Join(dataCallColumns, ", ")
)

type DataCall struct {
	DataCallID  int32     `json:"datacallid"`
	DataCall    string    `json:"datacall"`
	DateCreated time.Time `json:"datecreated"`
	Deadline    time.Time `json:"deadline"`
	// The questionnaire version the call answers (ztmf-misc#397). Chosen at
	// create, newest published unless named; changed only through
	// PUT /datacalls/{id}/questionnaire-version, which translates the answers.
	QuestionnaireVersionID int32 `json:"questionnaire_versionid" db:"questionnaire_versionid"`
}

func (d *DataCall) Save(ctx context.Context) (*DataCall, error) {

	// if valid, err := d.isValid(); !valid {
	// 	return err
	// }

	var (
		dataCall *DataCall
		err      error
	)
	if d.DataCallID == 0 {
		dataCall, err = d.create(ctx)
	} else {
		// The pin is not set here, whatever the body says: moving a call to
		// another version has to translate its answers, which only
		// PUT /datacalls/{id}/questionnaire-version does.
		dataCall, err = queryRow(ctx, stmntBuilder.
			Update("datacalls").
			Set("datacall", d.DataCall).
			Set("deadline", d.Deadline).
			Where("datacallid=?", d.DataCallID).
			Suffix("RETURNING "+dataCallColumnList), pgx.RowToStructByName[DataCall])
	}
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

// create inserts a call pinned to the version it names, or to the newest
// published one. The version is held FOR SHARE until commit, so it cannot be
// archived under a call that is about to pin it.
func (d *DataCall) create(ctx context.Context) (*DataCall, error) {
	sqlb := stmntBuilder.Insert("datacalls").Columns("datacall", "deadline", "questionnaire_versionid")
	res, err := scoreTx(ctx, func(tx pgx.Tx) (*DataCall, error) {
		version := d.QuestionnaireVersionID
		if version == 0 {
			// Shared with other creates; waits out a publish or archive so the
			// newest published version cannot change between this read and its lock.
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1)`, lifecycleLockKey); err != nil {
				return nil, trapError(err)
			}
			var current *int32
			if err := tx.QueryRow(ctx, `SELECT MAX(versionid) FROM questionnaire_versions WHERE state = 'published'`).Scan(&current); err != nil {
				return nil, trapError(err)
			}
			if current == nil {
				return nil, &InvalidInputError{data: map[string]any{"questionnaire_versionid": "no questionnaire version is published"}}
			}
			version = *current
		}
		if err := lockPinnableVersion(ctx, tx, version); err != nil {
			return nil, err
		}
		sqlb = sqlb.Values(d.DataCall, d.Deadline, version).Suffix("RETURNING " + dataCallColumnList)
		sql, args, _ := sqlb.ToSql()
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			return nil, trapError(err)
		}
		row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[DataCall])
		return &row, trapError(err)
	})
	if err != nil {
		return nil, err
	}
	recordEvent(ctx, sqlb, *res)
	return res, nil
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

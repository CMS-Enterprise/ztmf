package model

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

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
			Columns("notes", "functionoptionid", "datacallid").
			Values(s.Notes, s.FunctionOptionID, s.DataCallID).
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

	dataCall, err := FindDataCallByID(ctx, s.DataCallID)
	if err != nil {
		return err
	}

	if time.Now().UTC().After(dataCall.Deadline) {
		return ErrPastDeadline
	}

	return nil
}

type ScoreAggregate struct {
	DataCallID    int32   `json:"datacallid"`
	FismaSystemID int32   `json:"fismasystemid"`
	SystemScore   float64 `json:"systemscore"`
}

type FindScoresInput struct {
	input
	FismaSystemID  *int32 `schema:"fismasystemid"`
	FismaSystemIDs []*int32
	DataCallID     *int32 `schema:"datacallid"`
	UserID         *string
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

func FindScoresAggregate(ctx context.Context, input FindScoresInput) ([]*ScoreAggregate, error) {
	subSqlb := squirrel.Select("datacallid, fismasystemid, AVG(score) OVER (PARTITION BY datacallid, fismasystemid) as systemscore").
		From("scores").
		InnerJoin("functionoptions on functionoptions.functionoptionid=scores.functionoptionid")

	if input.DataCallID != nil {
		subSqlb = subSqlb.Where("datacallid=?", input.DataCallID)
	}

	if len(input.FismaSystemIDs) > 0 {
		subSqlb = subSqlb.Where(squirrel.Eq{"fismasystemid": input.FismaSystemIDs})
	}

	sqlb := squirrel.Select("*").
		FromSelect(subSqlb, "avg_by_datacall_fismasystem").
		GroupBy("datacallid, fismasystemid, systemscore").
		PlaceholderFormat(squirrel.Dollar)

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[ScoreAggregate])
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

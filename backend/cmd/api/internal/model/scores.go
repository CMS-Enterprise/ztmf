package model

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type Score struct {
	ScoreID          int32   `json:"scoreid"`
	FismaSystemID    int32   `json:"fismasystemid"`
	DateCalculated   float64 `json:"datecalculated"`
	Notes            *string `json:"notes"`
	FunctionOptionID int32   `json:"functionoptionid"`
	DataCallID       int32   `json:"datacallid"`
}

type ScoreAggregate struct {
	DataCallID    int32   `json:"datacallid"`
	FismaSystemID int32   `json:"fismasystemid"`
	SystemScore   float64 `json:"systemscore"`
}

type SaveScoreInput struct {
	ScoreID          *int32  `json:"scoreid"`
	FismaSystemID    int32   `json:"fismasystemid"`
	Notes            *string `json:"notes"`
	FunctionOptionID int32   `json:"functionoptionid"`
	DataCallID       int32   `json:"datacallid"`
}

type FindScoresInput struct {
	FismaSystemID  *int32
	FismaSystemIDs []*int32
	DataCallID     *int32
	UserID         *string
}

func FindScores(ctx context.Context, input FindScoresInput) ([]*Score, error) {
	sqlb := stmntBuilder.Select("scoreid, scores.fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid").From("scores")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid=scores.fismasystemid AND users_fismasystems.userid=?", *input.UserID)
	}

	if input.FismaSystemID != nil {
		sqlb = sqlb.Where("scores.fismasystemid=?", *input.FismaSystemID)
	}

	if input.DataCallID != nil {
		sqlb = sqlb.Where("datacallid=?", *input.DataCallID)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[Score])
}

func CreateScore(ctx context.Context, input SaveScoreInput) (*Score, error) {
	sqlb := stmntBuilder.Insert("public.scores").
		Columns("fismasystemid, notes, functionoptionid, datacallid").
		Values(input.FismaSystemID, input.Notes, input.FunctionOptionID, input.DataCallID).
		Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid")

	return queryRow(ctx, sqlb, pgx.RowToStructByName[Score])
}

func UpdateScore(ctx context.Context, input SaveScoreInput) error {
	if input.ScoreID == nil {
		return errors.New("input.ScoreID must be provided")
	}

	sqlb := stmntBuilder.Update("public.scores").
		Set("fismasystemid", &input.FismaSystemID).
		Set("notes", &input.Notes).
		Set("functionoptionid", &input.FunctionOptionID).
		Set("datacallid", &input.DataCallID).
		Where("scoreid=?", input.ScoreID)

	err := exec(ctx, sqlb)
	return err
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

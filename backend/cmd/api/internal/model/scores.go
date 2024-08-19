package model

import (
	"context"
	"errors"
	"log"

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

type SaveScoreInput struct {
	ScoreID          *int32  `json:"scoreid"`
	FismaSystemID    int32   `json:"fismasystemid"`
	Notes            *string `json:"notes"`
	FunctionOptionID int32   `json:"functionoptionid"`
	DataCallID       int32   `json:"datacallid"`
}

type FindScoresInput struct {
	FismaSystemID *int32
	DataCallID    *int32
	UserID        *string
}

func FindScores(ctx context.Context, input FindScoresInput) ([]*Score, error) {
	sqlb := sqlBuilder.Select("scoreid, scores.fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid").From("scores")

	if input.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid=scores.fismasystemid AND users_fismasystems.userid=?", *input.UserID)
	}

	if input.FismaSystemID != nil {
		sqlb = sqlb.Where("scores.fismasystemid=?", *input.FismaSystemID)
	}

	if input.DataCallID != nil {
		sqlb = sqlb.Where("datacallid=?", *input.DataCallID)
	}

	sql, boundArgs, _ := sqlb.ToSql()
	rows, err := query(ctx, sql, boundArgs...)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Score, error) {
		score := Score{}
		err := row.Scan(&score.ScoreID, &score.FismaSystemID, &score.DateCalculated, &score.Notes, &score.FunctionOptionID, &score.DataCallID)
		return &score, err
	})
}

func CreateScore(ctx context.Context, input SaveScoreInput) (*Score, error) {
	sqlb := sqlBuilder.Insert("public.scores").
		Columns("fismasystemid, notes, functionoptionid, datacallid").
		Values(input.FismaSystemID, input.Notes, input.FunctionOptionID, input.DataCallID).
		Suffix("RETURNING scoreid, fismasystemid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, notes, functionoptionid, datacallid")

	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, err
	}

	score := Score{}
	err = row.Scan(&score.ScoreID, &score.FismaSystemID, &score.DateCalculated, &score.Notes, &score.FunctionOptionID, &score.DataCallID)

	return &score, err
}

func UpdateScore(ctx context.Context, input SaveScoreInput) error {
	if input.ScoreID == nil {
		return errors.New("input.ScoreID must be provided")
	}

	sqlb := sqlBuilder.Update("public.scores").
		Set("fismasystemid", &input.FismaSystemID).
		Set("notes", &input.Notes).
		Set("functionoptionid", &input.FunctionOptionID).
		Set("datacallid", &input.DataCallID).
		Where("scoreid=?", input.ScoreID)

	sql, boundArgs, _ := sqlb.ToSql()
	err := exec(ctx, sql, boundArgs...)
	return err
}

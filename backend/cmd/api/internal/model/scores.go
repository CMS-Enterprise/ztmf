package model

import (
	"context"
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

// TODO: reimplement for REST
// func NewFunctionScore(ctx context.Context, fismasystemid int32, functionid int32, score float64, notes *string) (*FunctionScore, error) {
// 	sql := "INSERT INTO public.functionscores (fismasystemid, functionid, datecalculated, score, notes) VALUES ($1, $2, $3, $4, $5) RETURNING scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes"
// 	args := []any{fismasystemid, functionid, time.Now(), score, notes}
// 	return writeFunctionScore(ctx, sql, args)
// }

// func UpdateFunctionScore(ctx context.Context, scoreid *graphql.ID, fismasystemid int32, functionid int32, score float64, notes *string) (*FunctionScore, error) {
// 	sql := "UPDATE public.functionscores SET fismasystemid=$1, functionid=$2, datecalculated=$3, score=$4, notes=$5 WHERE scoreid=$6 RETURNING scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes"
// 	args := []any{fismasystemid, functionid, time.Now(), score, notes, scoreid}
// 	return writeFunctionScore(ctx, sql, args)
// }

// func writeFunctionScore(ctx context.Context, sql string, args []any) (*FunctionScore, error) {
// 	row, err := queryRow(ctx, sql, args...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	fs := FunctionScore{}
// 	err = row.Scan(&fs.Scoreid, &fs.Fismasystemid, &fs.Functionid, &fs.Datecalculated, &fs.Score, &fs.Notes)

// 	return &fs, err
// }

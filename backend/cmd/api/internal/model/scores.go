package model

import (
	"context"

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

func (s *Score) Save(ctx context.Context) (*Score, error) {
	var sqlb SqlBuilder

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
	return queryRow(ctx, sqlb, pgx.RowToStructByName[Score])
}

type ScoreAggregate struct {
	DataCallID    int32   `json:"datacallid"`
	FismaSystemID int32   `json:"fismasystemid"`
	SystemScore   float64 `json:"systemscore"`
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

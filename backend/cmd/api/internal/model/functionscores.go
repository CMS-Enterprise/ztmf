package model

import (
	"context"
	"log"
	"time"

	"github.com/graph-gophers/graphql-go"
)

type FunctionScore struct {
	Scoreid        graphql.ID
	Fismasystemid  int32
	Functionid     int32
	Datecalculated float64
	Score          float64
	Notes          *string
}

func (f *FunctionScore) Function(ctx context.Context) (*Function, error) {
	row, err := queryRow(context.Background(), "SELECT * FROM public.functions WHERE functionid=$1", f.Functionid)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	function := Function{}
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Datacenterenvironment)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &function, nil
}

func NewFunctionScore(ctx context.Context, fismasystemid int32, functionid int32, score float64, notes *string) (*FunctionScore, error) {
	sql := "INSERT INTO public.functionscores (fismasystemid, functionid, datecalculated, score, notes) VALUES ($1, $2, $3, $4, $5) RETURNING scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes"
	args := []any{fismasystemid, functionid, time.Now(), score, notes}
	return writeFunctionScore(ctx, sql, args)
}

func UpdateFunctionScore(ctx context.Context, scoreid *graphql.ID, fismasystemid int32, functionid int32, score float64, notes *string) (*FunctionScore, error) {
	sql := "UPDATE public.functionscores SET fismasystemid=$1, functionid=$2, datecalculated=$3, score=$4, notes=$5 WHERE scoreid=$6 RETURNING scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes"
	args := []any{fismasystemid, functionid, time.Now(), score, notes, scoreid}
	return writeFunctionScore(ctx, sql, args)
}

func writeFunctionScore(ctx context.Context, sql string, args []any) (*FunctionScore, error) {
	row, err := queryRow(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	fs := FunctionScore{}
	err = row.Scan(&fs.Scoreid, &fs.Fismasystemid, &fs.Functionid, &fs.Datecalculated, &fs.Score, &fs.Notes)

	return &fs, err
}

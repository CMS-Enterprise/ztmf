package engine

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
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

func (r *rootResolver) FunctionScores() ([]*FunctionScoreResolver, error) {
	var functionScoreRxs []*FunctionScoreResolver

	db := db.GetPool()

	rows, err := db.Query(context.Background(), "SELECT scoreid, fismasystemid, functionid, EXTRACT(EPOCH FROM datecalculated) as datecalculated, score, notes FROM functionscores ORDER BY scoreid ASC")
	if err != nil {
		log.Print(err)
		return nil, err
	}

	for rows.Next() {
		functionScore := FunctionScore{}
		err := rows.Scan(&functionScore.Scoreid, &functionScore.Fismasystemid, &functionScore.Functionid, &functionScore.Datecalculated, &functionScore.Score, &functionScore.Notes)
		if err != nil {
			log.Print(err)
		}
		functionRx := &FunctionScoreResolver{&functionScore}
		functionScoreRxs = append(functionScoreRxs, functionRx)
	}

	return functionScoreRxs, nil
}

type FunctionScoreResolver struct{ f *FunctionScore }

func (r *FunctionScoreResolver) Scoreid() graphql.ID {
	return r.f.Scoreid
}

func (r *FunctionScoreResolver) Fismasystemid() int32 {
	return r.f.Fismasystemid
}

func (r *FunctionScoreResolver) Functionid() int32 {
	return r.f.Functionid
}

func (r *FunctionScoreResolver) Datecalculated() float64 {
	return r.f.Datecalculated
}

func (r *FunctionScoreResolver) Score() float64 {
	return r.f.Score
}

func (r *FunctionScoreResolver) Notes() *string {
	if r.f.Notes == nil {
		s := ""
		return &s
	}
	return r.f.Notes
}

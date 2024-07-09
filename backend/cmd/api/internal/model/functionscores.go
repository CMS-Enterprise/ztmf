package model

import (
	"context"
	"log"

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
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &function, nil
}

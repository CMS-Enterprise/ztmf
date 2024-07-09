package model

import (
	"context"
	"log"

	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v5"
)

type Function struct {
	Functionid            graphql.ID
	Pillar                *string
	Name                  *string
	Description           *string
	Traditional           *string
	Initial               *string
	Advanced              *string
	Optimal               *string
	Datacenterenvironment *string
}

func FindFunctions(ctx context.Context) ([]*Function, error) {
	rows, err := query(ctx, "SELECT * FROM public.functions ORDER BY functionid ASC")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Function, error) {
		function := Function{}
		err := rows.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
		return &function, err
	})
}

func FindFunctionById(ctx context.Context, functionid graphql.ID) (*Function, error) {
	row, err := queryRow(ctx, "SELECT * FROM public.functions WHERE \"functionid\"=$1", functionid)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	function := Function{}
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &function, nil
}

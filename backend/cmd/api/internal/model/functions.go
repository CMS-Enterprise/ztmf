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
	Datacenterenvironment *string
	Questionid            *int32
}

func FindFunctions(ctx context.Context) ([]*Function, error) {
	rows, err := query(ctx, "SELECT * FROM public.functions ORDER BY functionid ASC")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Function, error) {
		log.Printf("%+v\n", row)
		function := Function{}
		err := row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Datacenterenvironment)
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
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Datacenterenvironment)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &function, nil
}

func (f *Function) Options(ctx context.Context) ([]*FunctionOption, error) {
	sql := "SELECT functionoptionid, functionid, score, optionname, description FROM public.functionoptions WHERE functionid=$1 ORDER BY score ASC"
	rows, err := query(ctx, sql, f.Functionid)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FunctionOption, error) {
		functionOption := FunctionOption{}
		err := row.Scan(&functionOption.Functionoptionid, &functionOption.Functionid, &functionOption.Score, &functionOption.Optionname, &functionOption.Description)
		return &functionOption, err
	})
}

func (f *Function) Question(ctx context.Context) (*Question, error) {
	row, err := queryRow(ctx, "SELECT * FROM questions WHERE questionid=$1", f.Questionid)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	question := Question{}
	err = row.Scan(&question.Questionid, &question.Question, &question.Notesprompt)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &question, nil
}

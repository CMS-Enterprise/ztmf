package main

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
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

func (r *rootResolver) Functions(ctx context.Context) ([]*FunctionResolver, error) {
	db, err := db.Conn(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	rows, err := db.Query(ctx, "SELECT * FROM public.functions ORDER BY functionid ASC")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*FunctionResolver, error) {
		function := Function{}
		err := rows.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
		return &FunctionResolver{&function}, err
	})

}

// resolver for graph entry from root
func (r *rootResolver) Function(ctx context.Context, args struct{ Functionid graphql.ID }) (*FunctionResolver, error) {
	db, err := db.Conn(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	row := db.QueryRow(context.Background(), "SELECT * FROM public.functions WHERE \"functionid\"=$1", string(args.Functionid))

	function := Function{}
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
	if err != nil {
		log.Println(err)
	}

	return &FunctionResolver{&function}, nil
}

// resolver for function score
func (r *FunctionScoreResolver) Function(ctx context.Context) (*FunctionResolver, error) {
	db, err := db.Conn(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	row := db.QueryRow(context.Background(), "SELECT * FROM public.functions WHERE \"functionid\"=$1", r.f.Functionid)

	function := Function{}
	err = row.Scan(&function.Functionid, &function.Pillar, &function.Name, &function.Description, &function.Traditional, &function.Initial, &function.Advanced, &function.Optimal, &function.Datacenterenvironment)
	if err != nil {
		log.Println(err)
	}

	return &FunctionResolver{&function}, nil
}

type FunctionResolver struct{ f *Function }

func (r *FunctionResolver) Functionid() graphql.ID {
	return r.f.Functionid
}

func (r *FunctionResolver) Pillar() *string {
	if r.f.Pillar == nil {
		s := ""
		return &s
	}
	return r.f.Pillar
}

func (r *FunctionResolver) Name() *string {
	if r.f.Name == nil {
		s := ""
		return &s
	}
	return r.f.Name
}

func (r *FunctionResolver) Description() *string {
	if r.f.Description == nil {
		s := ""
		return &s
	}
	return r.f.Description
}

func (r *FunctionResolver) Traditional() *string {
	if r.f.Traditional == nil {
		s := ""
		return &s
	}
	return r.f.Traditional
}

func (r *FunctionResolver) Initial() *string {
	if r.f.Initial == nil {
		s := ""
		return &s
	}
	return r.f.Initial
}

func (r *FunctionResolver) Advanced() *string {
	if r.f.Advanced == nil {
		s := ""
		return &s
	}
	return r.f.Advanced
}

func (r *FunctionResolver) Optimal() *string {
	if r.f.Optimal == nil {
		s := ""
		return &s
	}
	return r.f.Optimal
}

func (r *FunctionResolver) Datacenterenvironment() *string {
	if r.f.Datacenterenvironment == nil {
		s := ""
		return &s
	}
	return r.f.Datacenterenvironment
}

package engine

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v5"
)

type FismaSystem struct {
	Fismasystemid         graphql.ID
	Fismauid              string
	Fismaacronym          string
	Fismaname             string
	Fismasubsystem        *string
	Component             *string
	Groupacronym          *string
	Groupname             *string
	Divisionname          *string
	Datacenterenvironment *string
	Datacallcontact       *string
	Issoemail             *string
}

func (r *rootResolver) FismaSystems(args struct{ Fismaacronym *string }) ([]*FismaSystemResolver, error) {
	var fismaSystemsRxs []*FismaSystemResolver

	db := db.GetPool()
	sql := "SELECT * FROM fismasystems"
	if args.Fismaacronym != nil {
		sql += " WHERE fismaacronym=$1"
	}
	sql += " ORDER BY fismasystemid ASC"

	var rows pgx.Rows
	var err error
	if args.Fismaacronym != nil {
		rows, err = db.Query(context.Background(), sql, args.Fismaacronym)
	} else {
		rows, err = db.Query(context.Background(), sql)
	}

	if err != nil {
		log.Print(err)
		return nil, err
	}

	for rows.Next() {
		fismaSystem := FismaSystem{}
		rows.Scan(&fismaSystem.Fismasystemid, &fismaSystem.Fismauid, &fismaSystem.Fismaacronym, &fismaSystem.Fismaname, &fismaSystem.Fismasubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.Groupname, &fismaSystem.Divisionname, &fismaSystem.Datacenterenvironment, &fismaSystem.Datacallcontact, &fismaSystem.Issoemail)
		fismaSystemRx := FismaSystemResolver{&fismaSystem}
		fismaSystemsRxs = append(fismaSystemsRxs, &fismaSystemRx)
	}

	return fismaSystemsRxs, nil
}

func (r *rootResolver) FismaSystem(args struct{ Fismasystemid graphql.ID }) (*FismaSystemResolver, error) {
	// args.Fismasystemid
	db := db.GetPool()
	row := db.QueryRow(context.Background(), "SELECT * FROM fismasystems WHERE \"fismasystemid\"=$1", string(args.Fismasystemid))

	fismaSystem := FismaSystem{}
	err := row.Scan(&fismaSystem.Fismasystemid, &fismaSystem.Fismauid, &fismaSystem.Fismaacronym, &fismaSystem.Fismaname, &fismaSystem.Fismasubsystem, &fismaSystem.Component, &fismaSystem.Groupacronym, &fismaSystem.Groupname, &fismaSystem.Divisionname, &fismaSystem.Datacenterenvironment, &fismaSystem.Datacallcontact, &fismaSystem.Issoemail)
	if err != nil {
		log.Println(err)
	}

	return &FismaSystemResolver{&fismaSystem}, nil
}

type FismaSystemResolver struct{ f *FismaSystem }

func (r *FismaSystemResolver) Fismasystemid() graphql.ID {
	return r.f.Fismasystemid
}

func (r *FismaSystemResolver) Fismauid() string {
	return r.f.Fismauid
}

func (r *FismaSystemResolver) Fismaacronym() string {
	return r.f.Fismaacronym
}

func (r *FismaSystemResolver) Fismaname() string {
	return r.f.Fismaname
}

func (r *FismaSystemResolver) Fismasubsystem() *string {
	if r.f.Fismasubsystem == nil {
		s := ""
		return &s
	}
	return r.f.Fismasubsystem
}

func (r *FismaSystemResolver) Component() *string {
	if r.f.Component == nil {
		s := ""
		return &s
	}
	return r.f.Component
}

func (r *FismaSystemResolver) Groupacronym() *string {
	if r.f.Groupacronym == nil {
		s := ""
		return &s
	}
	return r.f.Groupacronym
}

func (r *FismaSystemResolver) Groupname() *string {
	if r.f.Groupname == nil {
		s := ""
		return &s
	}
	return r.f.Groupname
}

func (r *FismaSystemResolver) Divisionname() *string {
	if r.f.Divisionname == nil {
		s := ""
		return &s
	}
	return r.f.Divisionname
}

func (r *FismaSystemResolver) Datacenterenvironment() *string {
	if r.f.Datacenterenvironment == nil {
		s := ""
		return &s
	}
	return r.f.Datacenterenvironment
}

func (r *FismaSystemResolver) Datacallcontact() *string {
	if r.f.Datacallcontact == nil {
		s := ""
		return &s
	}
	return r.f.Datacallcontact
}

func (r *FismaSystemResolver) Issoemail() *string {
	if r.f.Issoemail == nil {
		s := ""
		return &s
	}
	return r.f.Issoemail
}
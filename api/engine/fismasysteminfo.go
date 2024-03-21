package engine

import (
	"context"
	"log"

	"github.com/graph-gophers/graphql-go"
)

type FismaSystem struct {
	FismaGUID             graphql.ID
	Fismaacronym          string
	Fismaname             string
	Fismasubsystem        string
	Component             string
	Groupacronym          string
	Groupname             string
	Divisionname          string
	Datacenterenvironment string
	Datacallcontact       string
	Issoemail             string
}

var fismaSystems = []*FismaSystem{
	{
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
		"test",
	},
}

func (r *rootResolver) FismaSystems() ([]*FismaSystemResolver, error) {
	var fismaSystemsRxs []*FismaSystemResolver

	db, _ := getDb()

	rows, err := db.Query(context.Background(), "SELECT * FROM public.fismasysteminfo")
	if err != nil {
		log.Print(err)
		return nil, err
	}

	for rows.Next() {
		var fismaGUID, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail string
		rows.Scan(&fismaGUID, &fismaacronym, &fismaname, &fismasubsystem, &component, &groupacronym, &groupname, &divisionname, &datacenterenvironment, &datacallcontact, &issoemail)
		fismaSystem := FismaSystem{graphql.ID(fismaGUID), fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail}
		fismaSystemRx := FismaSystemResolver{&fismaSystem}
		fismaSystemsRxs = append(fismaSystemsRxs, &fismaSystemRx)
	}

	return fismaSystemsRxs, nil
}

func (r *rootResolver) FismaSystem(args struct{ FismaGUID graphql.ID }) (*FismaSystemResolver, error) {
	// args.FismaGUID
	db, _ := getDb()
	row := db.QueryRow(context.Background(), "SELECT * FROM public.fismasysteminfo WHERE \"fismaGUID\"=$1", string(args.FismaGUID))

	var fismaGUID, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail string

	err := row.Scan(&fismaGUID, &fismaacronym, &fismaname, &fismasubsystem, &component, &groupacronym, &groupname, &divisionname, &datacenterenvironment, &datacallcontact, &issoemail)
	if err != nil {
		log.Println(err)
	}

	fismaSystem := FismaSystem{graphql.ID(fismaGUID), fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail}
	return &FismaSystemResolver{&fismaSystem}, nil
}

type FismaSystemResolver struct{ f *FismaSystem }

func (r *FismaSystemResolver) FismaGUID() graphql.ID {
	return r.f.FismaGUID
}

func (r *FismaSystemResolver) Fismaacronym() string {
	return r.f.Fismaacronym
}

func (r *FismaSystemResolver) Fismaname() string {
	return r.f.Fismaname
}

func (r *FismaSystemResolver) Fismasubsystem() string {
	return r.f.Fismasubsystem
}

func (r *FismaSystemResolver) Component() string {
	return r.f.Component
}

func (r *FismaSystemResolver) Groupacronym() string {
	return r.f.Groupacronym
}

func (r *FismaSystemResolver) Groupname() string {
	return r.f.Groupname
}

func (r *FismaSystemResolver) Divisionname() string {
	return r.f.Divisionname
}

func (r *FismaSystemResolver) Datacenterenvironment() string {
	return r.f.Datacenterenvironment
}

func (r *FismaSystemResolver) Datacallcontact() string {
	return r.f.Datacallcontact
}

func (r *FismaSystemResolver) Issoemail() string {
	return r.f.Issoemail
}

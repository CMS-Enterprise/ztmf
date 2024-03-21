package engine

import (
	"log"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type rootResolver struct{}

func GetHttpHandler() *relay.Handler {
	schema, err := graphql.ParseSchema(schema, &rootResolver{})
	if err != nil {
		log.Fatal(err)
	}

	return &relay.Handler{Schema: schema}
}

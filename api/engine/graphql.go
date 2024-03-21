package engine

import (
	"log"
	"os"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type rootResolver struct{}

func GetHttpHandler() *relay.Handler {
	rawSchemaBytes, err := os.ReadFile("schema.graphql") // just pass the file name
	if err != nil {
		log.Fatal(err)
	}

	schema, err := graphql.ParseSchema(string(rawSchemaBytes), &rootResolver{})
	if err != nil {
		log.Fatal(err)
	}

	return &relay.Handler{Schema: schema}
}

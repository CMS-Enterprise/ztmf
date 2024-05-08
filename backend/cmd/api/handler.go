package main

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type rootResolver struct{}

func HttpHandler() (*relay.Handler, error) {
	schema, err := graphql.ParseSchema(schema, &rootResolver{})
	if err != nil {
		return nil, err
	}

	return &relay.Handler{Schema: schema}, nil
}

package graphql

import (
	gql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

const rawSchema = `
type Query {
				hello: String!
}
`

type query struct{}

func (query) Hello() string { return "Hello, world!" }

func GetHttpHandler() *relay.Handler {
	var schema *gql.Schema = gql.MustParseSchema(rawSchema, &query{})
	return &relay.Handler{Schema: schema}
}

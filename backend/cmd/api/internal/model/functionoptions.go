package model

import "github.com/graph-gophers/graphql-go"

type FunctionOption struct {
	Functionoptionid graphql.ID
	Functionid       int32
	Score            int32
	Optionname       string
	Description      string
}

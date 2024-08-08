package model

import "github.com/graph-gophers/graphql-go"

type Question struct {
	Questionid  graphql.ID
	Question    string
	Notesprompt string
}

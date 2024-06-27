package main

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/token"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type rootResolver struct{}

func HttpHandler() (http.Handler, error) {
	schema, err := graphql.ParseSchema(schema, &rootResolver{})
	if err != nil {
		return nil, err
	}

	return recordUser(logRequest(&relay.Handler{Schema: schema})), nil
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: log requests
		next.ServeHTTP(w, r)
	})
}

func recordUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userContext, ok := r.Header[http.CanonicalHeaderKey("x-amzn-ava-user-context")]; ok {
			// fmt.Printf("x-amzn-ava-user-context: %s\n", userContext[0])
			token, _ := token.Decode(userContext[0])
			if !token.Valid {
				fmt.Println("token invalid")
			}
			fmt.Printf("%+v\n", token.Claims)
		}

		next.ServeHTTP(w, r)
	})
}

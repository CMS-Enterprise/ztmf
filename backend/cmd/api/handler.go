package main

import (
	"log"
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

	return recordUser(&relay.Handler{Schema: schema}), nil
}

func recordUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userContext, ok := r.Header[http.CanonicalHeaderKey("x-amzn-ava-user-context")]; ok {
			// fmt.Printf("x-amzn-ava-user-context: %s\n", userContext[0])
			tkn, err := token.Decode(userContext[0])
			if !tkn.Valid {
				w.WriteHeader(403)
				if claims, ok := tkn.Claims.(*token.Claims); ok {
					log.Printf("Invalid token received for %+v with error %s", claims, err)
				} else {
					log.Printf("Invalid token received with error %s", err)
				}
				// return now so we send nothing and don't complete the request for data!
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

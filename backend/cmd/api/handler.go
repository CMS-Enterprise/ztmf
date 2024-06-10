package main

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

type rootResolver struct{}

var tlsConstants = map[uint16]string{
	0x1301: "TLS_AES_128_GCM_SHA256",
	0x1302: "TLS_AES_256_GCM_SHA384",
	0x1303: "TLS_CHACHA20_POLY1305_SHA256",
	0x0301: "VersionTLS10",
	0x0302: "VersionTLS11",
	0x0303: "VersionTLS12",
	0x0304: "VersionTLS13",
}

func HttpHandler() (http.Handler, error) {
	schema, err := graphql.ParseSchema(schema, &rootResolver{})
	if err != nil {
		return nil, err
	}

	return logRequest(&relay.Handler{Schema: schema}), nil
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s\r", tlsConstants[r.TLS.Version], tlsConstants[r.TLS.CipherSuite])
		next.ServeHTTP(w, r)
	})
}

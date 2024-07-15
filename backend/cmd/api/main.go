package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/graph"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func main() {
	log.SetFlags(0)
	cfg := config.GetInstance()
	log.Println("Parsing schema...")

	schema, err := graphql.ParseSchema(graph.Schema, &graph.RootResolver{}, graphql.UseFieldResolvers())
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", auth.Middleware(&relay.Handler{Schema: schema}))
	mux.Handle("/whoami", auth.Middleware(auth.WhoAmI()))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Printf("%s environment listening on %s\n", cfg.Env, cfg.Port)

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		log.Print("Loading TLS configuration")
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Fatal(err)
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
			},
			MinVersion: tls.VersionTLS13,
		}
		log.Fatal("Failed to start server:", server.ListenAndServeTLS("", ""))

	} else {
		log.Fatal("Failed to start server:", server.ListenAndServe())
	}

}

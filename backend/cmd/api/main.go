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

	log.Printf("%s environment listening on %s\n", cfg.Env, cfg.Port)

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		log.Print("Loading TLS configuration")
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
			},
			MinVersion: tls.VersionTLS13,
		}

		server := &http.Server{
			Addr:      ":" + cfg.Port,
			Handler:   auth.Middleware(&relay.Handler{Schema: schema}),
			TLSConfig: tlsConfig,
		}
		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}

	} else {
		http.Handle("/graphql", auth.Middleware(&relay.Handler{Schema: schema}))
		log.Fatal("Failed to start server:", http.ListenAndServe(":"+cfg.Port, nil))
	}

}

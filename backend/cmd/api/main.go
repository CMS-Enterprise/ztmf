package main

import (
	"crypto/tls"
	_ "crypto/tls/fipsonly"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
)

func main() {

	cfg := config.GetInstance()

	handler, err := HttpHandler()
	if err != nil {
		log.Fatal("could not get HttpHandler()", err)
	}

	http.Handle("/graphql", handler)

	log.Printf("%s environment listening on %s\n", cfg.Env, cfg.Port)

	if cfg.CertFile != "" && cfg.KeyFile != "" {
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
			Addr:      cfg.Port,
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}

	} else {
		log.Fatal("could not listen and serve:", http.ListenAndServe(":"+cfg.Port, nil))
	}
}

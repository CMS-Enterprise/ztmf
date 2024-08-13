package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/router"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
)

func main() {
	log.SetFlags(0)
	cfg := config.GetInstance()

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router.Handler(),
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

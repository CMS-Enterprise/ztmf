package main

import (
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
		log.Fatal("could not listen and serve:", http.ListenAndServeTLS(":"+cfg.Port, cfg.CertFile, cfg.KeyFile, nil))
	} else {
		log.Fatal("could not listen and serve:", http.ListenAndServe(":"+cfg.Port, nil))
	}
}

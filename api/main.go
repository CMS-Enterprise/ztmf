package main

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/api/config"
	"github.com/CMS-Enterprise/ztmf/api/engine"
)

func main() {

	cfg := config.GetConfig()

	http.Handle("/graphql", engine.GetHttpHandler())

	log.Printf("%s environment listening on %s\n", cfg.ENV, cfg.PORT)
	switch cfg.ENV {
	case "local":
		log.Fatal(http.ListenAndServe(":"+cfg.PORT, nil))
	case "dev", "prod":
		log.Fatal(http.ListenAndServeTLS(":"+cfg.PORT, cfg.CERT_FILE, cfg.KEY_FILE, nil))
	}

}

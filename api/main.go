package main

import (
	"log"
	"net/http"

	"github.com/caarlos0/env/v10"

	"github.com/CMS-Enterprise/ztmf/api/graphql"
)

type Config struct {
	ENV       string `env:"ENVIRONMENT" envDefault:"local"`
	PORT      string `env:"PORT" envDefault:"3000"`
	CERT_FILE string `env:"CERT_FILE"`
	KEY_FILE  string `env:"KEY_FILE"`
}

func main() {

	var err error

	cfg := Config{}
	err = env.Parse(&cfg)
	if err != nil {
		log.Fatalf("unable to parse ennvironment variables: %e", err)
	}

	http.Handle("/query", graphql.GetHttpHandler())

	log.Printf("%s environment listening on %s\n", cfg.ENV, cfg.PORT)

	switch cfg.ENV {
	case "local":
		log.Fatal(http.ListenAndServe(":"+cfg.PORT, nil))
	case "dev", "prod":
		log.Fatal(http.ListenAndServeTLS(":"+cfg.PORT, cfg.CERT_FILE, cfg.KEY_FILE, nil))
	}
}

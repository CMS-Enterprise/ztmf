package config

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/caarlos0/env/v10"
)

type dbconfig struct {
	User     string `json:"username"`
	Pass     string `json:"password"`
	RawCreds string `env:"DB_CREDS"`
	Database string `env:"DB_NAME"`
	Endpoint string `env:"DB_ENDPOINT"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
}

type config struct {
	ENV       string `env:"ENVIRONMENT" envDefault:"local"`
	PORT      string `env:"PORT" envDefault:"3000"`
	CERT_FILE string `env:"CERT_FILE"`
	KEY_FILE  string `env:"KEY_FILE"`
	DB        *dbconfig
}

func (c *config) GetCorsOrigin() string {
	switch c.ENV {
	case "local":
		return "http://localhost"
	case "dev":
		return "https://dev.ztmf.cms.gov"
	case "prod":
		return "https://ztmf.cms.gov"
	}
	return ""
}

var cfg *config

func GetConfig() *config {

	if cfg == nil {
		log.Print("Initializing config...")

		var once sync.Once
		once.Do(func() {
			cfg = &config{DB: &dbconfig{}}
			err := env.Parse(cfg)
			if err != nil {
				log.Fatalf("unable to parse environment variables: %e", err)
			}

			err = json.Unmarshal([]byte(cfg.DB.RawCreds), cfg.DB)
			if err != nil {
				log.Fatalf("unable to parse db creds: %e", err)
			}
		})
	}

	return cfg
}

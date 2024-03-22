package engine

import (
	"encoding/json"
	"log"

	"github.com/caarlos0/env/v10"
)

type dbconfig struct {
	User     string `json:"username"`
	Pass     string `json:"password"`
	RawCreds string `env:"DB_CREDS" envDefault:"{\"username\":\"none\",\"password\":\"none\"}"`
	Database string `env:"DB_NAME"`
	Endpoint string `env:"DB_ENDPOINT" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
}

type Config struct {
	ENV       string `env:"ENVIRONMENT" envDefault:"local"`
	PORT      string `env:"PORT" envDefault:"3000"`
	CERT_FILE string `env:"CERT_FILE"`
	KEY_FILE  string `env:"KEY_FILE"`
	DB        dbconfig
}

var cfg Config

func GetConfig() Config {

	if cfg.ENV == "" {
		log.Print("Initializing config...")
		err := env.Parse(&cfg)
		if err != nil {
			log.Fatalf("unable to parse ennvironment variables: %e", err)
		}
	}
	json.Unmarshal([]byte(cfg.DB.RawCreds), &cfg.DB)
	return cfg
}

package config

import (
	"log"
	"sync"

	"github.com/caarlos0/env/v10"
)

// config is shared by all binaries with values derived from environment variables
type config struct {
	Env      string `env:"ENVIRONMENT" envDefault:"local"`
	Port     string `env:"PORT" envDefault:"3000"`
	CertFile string `env:"CERT_FILE"`
	KeyFile  string `env:"KEY_FILE"`
	Region   string `env:"AWS_REGION" envDefault:"us-east-1"`
	Auth     struct {
		HS256_SECRET string `env:"AUTH_HS256_SECRET"`
		TokenKeyUrl  string `env:"AUTH_TOKEN_KEY_URL"` // where to find the key that validates JWT
		HeaderField  string `env:"AUTH_HEADER_FIELD"`  // the header that includes encoded JWT from OIDC IDP
	}
	Db struct {
		Host        string  `env:"DB_ENDPOINT"`
		Port        string  `env:"DB_PORT" envDefault:"5432"`
		Name        string  `env:"DB_NAME"`
		User        string  `env:"DB_USER"`
		Pass        string  `env:"DB_PASS"`
		SecretId    string  `env:"DB_SECRET_ID"`
		PopulateSql *string `env:"DB_POPULATE"` // path to sql to populate test database
	}
}

var (
	cfg  *config
	once sync.Once
)

// GetInstance returns a singleton of *config
func GetInstance() *config {
	if cfg == nil {
		var err error
		log.Println("Initializing config...")

		once.Do(func() {
			cfg = &config{}
			err = env.Parse(cfg)
		})
		if err != nil {
			log.Fatal("could not parse environment variables", err)
			return nil
		}
	}

	return cfg
}

package config

import (
	"crypto/x509"
	"errors"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
	"github.com/caarlos0/env/v10"
)

var cfg *config

type smtp struct {
	User     string `json:"user" env:"SMTP_USER"`
	Pass     string `json:"pass" env:"SMTP_PASS"`
	Host     string `json:"host" env:"SMTP_HOST"`
	Port     int16  `json:"port" env:"SMTP_PORT"`
	From     string `json:"from" env:"SMTP_FROM"`
	TestMode bool   `json:"-" env:"SMTP_TEST_MODE"`
	// certs is a chain comprised of root and intermediate certificates pulled from secrets manager
	Certs                    *x509.CertPool
	ConfigSecretID           *string `env:"SMTP_CONFIG_SECRET_ID"`
	CertRootSecretID         *string `env:"SMTP_CA_ROOT_SECRET_ID"`
	CertIntermediateSecretID *string `env:"SMTP_CA_INT_SECRET_ID"`
}

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
	// SMTP config will be loaded from env vars if provided.
	// If config secret is provided, struct field values will be overwritten by unmarshalling JSON from config secret value hence the pointer to struct
	SMTP *smtp
}

// GetInstance returns a singleton of *config
func GetInstance() *config {
	if cfg == nil {
		var (
			err  error
			once sync.Once
		)

		once.Do(func() {
			var (
				smtpCfgSecret, SmtpCertRootSecret, SmtpCertIntermediateSecret *secrets.Secret
				secretVal                                                     *string
			)

			log.Println("initializing config...")

			cfg = &config{
				SMTP: &smtp{},
			}
			err = env.Parse(cfg)
			if err != nil {
				log.Println("error parsing environment variables: ", err)
				return
			}

			if cfg.SMTP.ConfigSecretID != nil {
				smtpCfgSecret, err = secrets.NewSecret(*cfg.SMTP.ConfigSecretID)
				if err != nil {
					return
				}

				err = smtpCfgSecret.Unmarshal(cfg.SMTP)
				if err != nil {
					return
				}
			}

			if cfg.SMTP.CertRootSecretID != nil && cfg.SMTP.CertIntermediateSecretID != nil {
				cfg.SMTP.Certs = x509.NewCertPool()

				SmtpCertRootSecret, err = secrets.NewSecret(*cfg.SMTP.CertRootSecretID)
				if err != nil {
					return
				}

				secretVal, err = SmtpCertRootSecret.Value()
				if err != nil {
					return
				}

				if !cfg.SMTP.Certs.AppendCertsFromPEM([]byte(*secretVal)) {
					err = errors.New("failed to append root cert")
					return
				}

				SmtpCertIntermediateSecret, err = secrets.NewSecret(*cfg.SMTP.CertIntermediateSecretID)
				if err != nil {
					return
				}

				secretVal, err = SmtpCertIntermediateSecret.Value()
				if err != nil {
					return
				}

				if !cfg.SMTP.Certs.AppendCertsFromPEM([]byte(*secretVal)) {
					err = errors.New("failed to append intermediate cert")
					return
				}
			}
		})

		if err != nil {
			// anything depending on the config instance can't possibly work if initialization failed, so exit
			log.Fatal("failed to initialize config: ", err)
			return nil
		}
	}

	return cfg
}

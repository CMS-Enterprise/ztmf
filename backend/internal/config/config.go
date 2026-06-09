package config

import (
	"context"
	"crypto/x509"
	"errors"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
	"github.com/caarlos0/env/v10"
)

var cfg *config

type smtp struct {
	User string `json:"user" env:"SMTP_USER"`
	Pass string `json:"pass" env:"SMTP_PASS"`
	Host string `json:"host" env:"SMTP_HOST"`
	Port int16  `json:"port" env:"SMTP_PORT"`
	From string `json:"from" env:"SMTP_FROM"`
	// certs is a chain comprised of root and intermediate certificates pulled from secrets manager
	Certs                    *x509.CertPool
	ConfigSecretID           *string `env:"SMTP_CONFIG_SECRET_ID"`
	CertRootSecretID         *string `env:"SMTP_CA_ROOT_SECRET_ID"`
	CertIntermediateSecretID *string `env:"SMTP_CA_INT_SECRET_ID"`
}

// config is shared by all binaries with values derived from environment variables
type config struct {
	Env      string `env:"ENVIRONMENT" envDefault:"production"`
	Port     string `env:"PORT" envDefault:"3000"`
	CertFile string `env:"CERT_FILE"`
	KeyFile  string `env:"KEY_FILE"`
	Region   string `env:"AWS_REGION" envDefault:"us-east-1"`
	Auth     struct {
		HS256_SECRET string `env:"AUTH_HS256_SECRET"`
		TokenKeyUrl  string `env:"AUTH_TOKEN_KEY_URL"` // where to find the key that validates JWT
		HeaderField  string `env:"AUTH_HEADER_FIELD"`  // the header that includes encoded JWT from OIDC IDP

		// OktaIssuer is the expected iss claim for tokens minted by the CMS Okta
		// IdP. When set, IdP tokens whose issuer does not match an allowed issuer
		// are rejected. Left empty in local dev (HS256) where no issuer is asserted.
		OktaIssuer string `env:"AUTH_OKTA_ISSUER"`

		// Entra* configure the second (HHS) OIDC provider. EntraIssuer is the
		// exact iss claim (the /v2.0 suffix is part of the issuer, not optional).
		// EntraJWKSUrl is the key set used to verify RS256 signatures.
		// EntraTenantID is pinned against the token tid claim so that only the
		// HHS tenant can authenticate, even if another Entra tenant presents a
		// validly-signed token.
		EntraIssuer   string `env:"AUTH_ENTRA_ISSUER"`
		EntraJWKSUrl  string `env:"AUTH_ENTRA_JWKS_URL"`
		EntraTenantID string `env:"AUTH_ENTRA_TENANT_ID"`

		// SessionSigningSecret signs the application session JWT minted after a
		// successful IdP login (Option A: ALB stops gating /api/*, the backend
		// gates it instead). Falls back to HS256_SECRET when unset so local dev
		// and the E2E suite, which send an HS256 bearer directly, keep working.
		SessionSigningSecret string `env:"AUTH_SESSION_SIGNING_SECRET"`
		// SessionCookieName is the cookie that carries the app session token.
		SessionCookieName string `env:"AUTH_SESSION_COOKIE_NAME" envDefault:"ztmf_session"`
		// SessionTTL is the app session lifetime in seconds.
		SessionTTL int `env:"AUTH_SESSION_TTL" envDefault:"10800"`
		// CookieDomain scopes the session cookie (e.g. dev.ztmf.cms.gov). Empty
		// scopes the cookie to the exact request host, which is correct locally.
		CookieDomain string `env:"AUTH_COOKIE_DOMAIN"`
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

				secretVal, err = SmtpCertRootSecret.Value(context.Background())
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

				secretVal, err = SmtpCertIntermediateSecret.Value(context.Background())
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

// SessionSecret returns the secret used to sign and verify the application
// session token. It prefers AUTH_SESSION_SIGNING_SECRET and falls back to the
// HS256 secret so that local dev and the E2E suite, which present an HS256
// bearer directly rather than going through the ALB OIDC login, continue to
// authenticate without extra configuration.
func (c *config) SessionSecret() []byte {
	if c.Auth.SessionSigningSecret != "" {
		return []byte(c.Auth.SessionSigningSecret)
	}
	return []byte(c.Auth.HS256_SECRET)
}

// IsLocal reports whether the API is running in the local development
// environment (ENVIRONMENT=local). Used to gate dev-only behavior such as
// just-in-time user creation, which must not happen in any other environment.
func (c *config) IsLocal() bool {
	return c.Env == "local"
}

// IsLocalOrTest reports whether the API is running in an ephemeral local or
// E2E test environment (ENVIRONMENT=local or test). Used to gate test-data
// seeding, which is safe in both but must never run against a deployed
// environment. Kept distinct from IsLocal because seeding applies to the E2E
// test stack while just-in-time user creation deliberately does not.
func (c *config) IsLocalOrTest() bool {
	return c.Env == "local" || c.Env == "test"
}

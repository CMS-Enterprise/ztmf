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

// DbCreds holds resolved database credentials, either read directly from env
// vars or unmarshalled from the Secrets Manager secret named by DB_SECRET_ID.
type DbCreds struct {
	Username string
	Password string
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

		// Entra* configure the second OIDC provider. EntraIssuer is the exact
		// iss claim (the /v2.0 suffix is part of the issuer, not optional).
		// EntraJWKSUrl is the key set used to verify RS256 signatures.
		// EntraTenantID is pinned against the token tid claim so that only the
		// configured tenant can authenticate, even if another Entra tenant
		// presents a validly-signed token.
		EntraIssuer   string `env:"AUTH_ENTRA_ISSUER"`
		EntraJWKSUrl  string `env:"AUTH_ENTRA_JWKS_URL"`
		EntraTenantID string `env:"AUTH_ENTRA_TENANT_ID"`

		// OktaAudience / EntraAudience are the expected aud claim (the ZTMF app's
		// client id / api identifier) for each IdP. When set, a token whose aud
		// does not contain the expected value is rejected, so a validly-signed
		// token minted for a *different* application in the same issuer/tenant
		// cannot authenticate to ZTMF. Optional and enforced only when set, in
		// parallel with EntraTenantID pinning: empty means no audience check, which
		// preserves local/dev (HS256, no aud) and the legacy single-app behavior.
		OktaAudience  string `env:"AUTH_OKTA_AUDIENCE"`
		EntraAudience string `env:"AUTH_ENTRA_AUDIENCE"`

		// SessionSigningSecret signs the application session JWT minted after a
		// successful IdP login (Option A: ALB stops gating /api/*, the backend
		// gates it instead). Falls back to HS256_SECRET when unset so local dev
		// and the E2E suite, which send an HS256 bearer directly, keep working.
		SessionSigningSecret string `env:"AUTH_SESSION_SIGNING_SECRET"`
		// SessionCookieName is the cookie that carries the app session token.
		SessionCookieName string `env:"AUTH_SESSION_COOKIE_NAME" envDefault:"ztmf_session"`
		// SessionTTL is the app session lifetime in seconds.
		SessionTTL int `env:"AUTH_SESSION_TTL" envDefault:"10800"`
		// OriginHost is the host the same-origin check expects in the Origin
		// (then Referer) header of a state-changing request, e.g.
		// dev.ztmf.cms.gov. Empty compares against the request Host instead,
		// which is what local dev and the test suite rely on. This does not
		// affect cookie scope: the session cookie is always host-only.
		OriginHost string `env:"AUTH_ORIGIN_HOST"`
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

	// dbSecret caches the Secrets Manager secret behind Db.SecretId. Unexported
	// (env.Parse skips it) and per-instance so tests can exercise resolution on a
	// constructed config without touching the singleton.
	dbSecret     *secrets.Secret
	dbSecretOnce sync.Once
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

// DbCreds resolves the current database credentials. If no secret id is
// specified, user/pass are assumed to be provided in env vars; otherwise they
// are pulled from the secret. Call dbSecretOnce.Do unconditionally (no bare
// dbSecret nil-check first): once.Do is what establishes the happens-before for
// the dbSecret write, so reading the pointer outside it - from concurrent
// request goroutines at startup - would be an unsynchronized read. err is set
// only on the goroutine that ran the init, so re-check dbSecret afterward to
// surface a failed init to every caller.
func (c *config) DbCreds() (*DbCreds, error) {
	if c.Db.SecretId == "" {
		return &DbCreds{c.Db.User, c.Db.Pass}, nil
	}

	var err error
	c.dbSecretOnce.Do(func() {
		c.dbSecret, err = secrets.NewSecret(c.Db.SecretId)
	})
	if err != nil {
		return nil, err
	}
	if c.dbSecret == nil {
		return nil, errors.New("db secret initialization failed")
	}

	creds := &DbCreds{}
	if err := c.dbSecret.Unmarshal(creds); err != nil {
		log.Println("could not unmarshal credentials", err)
		return nil, err
	}

	return creds, nil
}

// RefreshDbCreds forces a re-read of the cached secret from Secrets Manager so
// the next connection picks up the post-rotation password. Caches the secret on
// first use if startup somehow skipped it.
func (c *config) RefreshDbCreds(ctx context.Context) error {
	if c.dbSecret == nil {
		if _, err := c.DbCreds(); err != nil {
			return err
		}
		if c.dbSecret == nil {
			return errors.New("no db secret configured to refresh")
		}
	}
	return c.dbSecret.Refresh(ctx)
}

// SessionSecret returns the secret used to sign and verify the application
// session token. It prefers AUTH_SESSION_SIGNING_SECRET. The fallback to the
// HS256 secret is allowed ONLY in local/test, where that secret is the
// well-known dev value and sessions are minted from HS256 bearers anyway. In a
// deployed environment the fallback is refused: if AUTH_SESSION_SIGNING_SECRET
// is unset the function returns nil so MintSession/ParseSession fail closed,
// rather than silently signing production sessions with a shared HS256 key.
func (c *config) SessionSecret() []byte {
	if c.Auth.SessionSigningSecret != "" {
		return []byte(c.Auth.SessionSigningSecret)
	}
	if c.IsLocalOrTest() {
		return []byte(c.Auth.HS256_SECRET)
	}
	return nil
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

package db

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	once     sync.Once
	dbSecret *secrets.Secret
)

type dbCreds struct {
	Username string
	Password string
}

// Conn opens a fresh connection to the database. RDS-managed credentials in
// Secrets Manager auto-rotate (7-day schedule); the secret we cache at startup
// goes stale at each rotation and the next connect fails SASL auth (28P01).
// Rather than let that take the process down, an auth failure forces a re-fetch
// of the secret and a single retry, so a routine rotation is ridden out in
// place instead of relying on an ECS task restart to recover. A genuinely wrong
// password (not a rotation) fails the retry too and surfaces as a normal error.
func Conn(ctx context.Context) (*pgx.Conn, error) {
	conn, err := connect(ctx)
	if err == nil {
		return conn, nil
	}

	// Only the secret-backed path can recover by refreshing; local env-var
	// credentials have nothing to re-fetch, so do not retry there.
	if !isAuthError(err) || config.GetInstance().Db.SecretId == "" {
		return nil, err
	}

	log.Println("db authentication failed; refreshing credentials and retrying once")
	if rerr := refreshDbCreds(ctx); rerr != nil {
		log.Println("could not refresh db credentials", rerr)
		return nil, err
	}

	return connect(ctx)
}

// connect builds the connection config from the current credentials and opens a
// single connection. Split out from Conn so the credential-refresh retry can
// re-run the whole path against the freshly fetched secret.
func connect(ctx context.Context) (*pgx.Conn, error) {
	creds, err := getDbCreds()
	if err != nil {
		log.Println("could not get db credentials")
		return nil, err
	}

	cfg := config.GetInstance()
	connConfig, err := pgx.ParseConfig("host=" + cfg.Db.Host + " port=" + cfg.Db.Port + " dbname=" + cfg.Db.Name + " user=" + creds.Username + " password=" + creds.Password + " sslmode=allow")
	if err != nil {
		log.Println("could not parse db config", err)
		return nil, err
	}

	// Surface PostgreSQL NOTICE messages (e.g. RAISE NOTICE in migration DO
	// blocks) in the application log instead of the silent default. Without
	// this, migration diagnostics only land in the RDS PostgreSQL log group
	// in CloudWatch, which is not where deploys are normally watched.
	connConfig.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		log.Printf("pg notice [%s]: %s", n.Severity, n.Message)
	}

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		log.Println("could not connect to db", err)
		return nil, err
	}

	return conn, nil
}

// isAuthError reports whether err is a PostgreSQL password-authentication
// failure (SQLSTATE 28P01), the signature of a connection opened with
// credentials that no longer match - e.g. a rotated-away cached secret.
func isAuthError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "28P01"
}

// refreshDbCreds forces a re-read of the cached secret from Secrets Manager so
// the next connect picks up the post-rotation password. Caches the secret on
// first use if startup somehow skipped it.
func refreshDbCreds(ctx context.Context) error {
	if dbSecret == nil {
		if _, err := getDbCreds(); err != nil {
			return err
		}
		if dbSecret == nil {
			return errors.New("no db secret configured to refresh")
		}
	}
	return dbSecret.Refresh(ctx)
}

func getDbCreds() (*dbCreds, error) {
	cfg := config.GetInstance()

	// TODO: move this secret handling to config.GetInstance()
	// if no secret id specified, assume user/pass are provided in env vars
	if cfg.Db.SecretId == "" {
		return &dbCreds{cfg.Db.User, cfg.Db.Pass}, nil
	}

	// otherwise pull user/pass from the secret
	var err error
	if dbSecret == nil {
		once.Do(func() {
			dbSecret, err = secrets.NewSecret(cfg.Db.SecretId)
		})
		if err != nil {
			return nil, err
		}
	}

	creds := &dbCreds{}
	err = dbSecret.Unmarshal(creds)
	if err != nil {
		log.Println("could not unmarshal credentials", err)
		return nil, err
	}

	return creds, nil
}

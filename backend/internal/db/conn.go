package db

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	once     sync.Once
	dbSecret *secrets.Secret

	pool     *pgxpool.Pool
	poolOnce sync.Once
	poolErr  error
)

// Pool bounds the number of open connections and, crucially, the maximum the
// cluster ever sees. A 0.5-1.0 ACU Aurora Serverless instance allows only a
// small number of connections, and the auth handshake (TLS + SCRAM) is
// CPU-heavy server-side. MaxConns is therefore set well under that ceiling.
const (
	maxConns        = 16
	minConns        = 2
	maxConnLifetime = 50 * time.Minute
	maxConnIdleTime = 5 * time.Minute
	healthCheck     = time.Minute
)

type dbCreds struct {
	Username string
	Password string
}

// Conn acquires a connection from the shared pool. Callers MUST call
// conn.Release() (typically deferred) to return it to the pool when done;
// releasing reuses the connection rather than tearing it down, which is the
// whole point of the pool. The previous implementation opened a brand-new
// connection per call, so under any concurrency the cluster's connection limit
// was exhausted and new connections died mid-handshake. The pool caps and
// reuses connections, so a burst of concurrent queries no longer opens a
// connection each.
//
// RDS-managed credentials rotate on a 7-day schedule; the secret cached at
// startup goes stale at each rotation and a connection opened with it fails
// SASL auth (28P01). Acquiring a connection can surface that error when the
// pool opens a new connection, so an auth failure refreshes the secret and
// retries once. The pool supplies credentials per new connection (see
// BeforeConnect), so the retry picks up the refreshed secret.
func Conn(ctx context.Context) (*pgxpool.Conn, error) {
	p, err := Pool(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := p.Acquire(ctx)
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

	conn, aerr := p.Acquire(ctx)
	if aerr != nil {
		log.Println("db connection still failing after credential refresh", aerr)
		return nil, aerr
	}
	return conn, nil
}

// Pool lazily builds the shared connection pool and returns it. The pool is
// built once for the process; subsequent calls return the same pool.
func Pool(ctx context.Context) (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		poolErr = initPool()
	})
	if poolErr != nil {
		return nil, poolErr
	}
	return pool, nil
}

func initPool() error {
	cfg := config.GetInstance()

	poolCfg, err := pgxpool.ParseConfig("host=" + cfg.Db.Host + " port=" + cfg.Db.Port + " dbname=" + cfg.Db.Name + " sslmode=allow")
	if err != nil {
		log.Println("could not parse db pool config", err)
		return err
	}

	poolCfg.MaxConns = maxConns
	poolCfg.MinConns = minConns
	poolCfg.MaxConnLifetime = maxConnLifetime
	poolCfg.MaxConnIdleTime = maxConnIdleTime
	poolCfg.HealthCheckPeriod = healthCheck

	// Supply credentials per new connection rather than baking them into the
	// connection string, so each connection the pool opens uses the current
	// secret value. Combined with MaxConnLifetime recycling connections, a
	// rotated credential is picked up without a process restart.
	poolCfg.BeforeConnect = func(_ context.Context, cc *pgx.ConnConfig) error {
		creds, err := getDbCreds()
		if err != nil {
			return err
		}
		cc.User = creds.Username
		cc.Password = creds.Password
		return nil
	}

	// Surface PostgreSQL NOTICE messages (e.g. RAISE NOTICE in migration DO
	// blocks) in the application log instead of the silent default.
	poolCfg.ConnConfig.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		log.Printf("pg notice [%s]: %s", n.Severity, n.Message)
	}

	// Build the pool with a background context, not the first caller's request
	// context: pool creation happens once for the process, and a request that
	// is canceled mid-init must not poison the pool for every later caller.
	pool, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Println("could not create db pool", err)
		return err
	}
	return nil
}

// MigrationConn opens a single dedicated connection for the schema migrator.
// The tern Migrator owns a *pgx.Conn for the lifetime of the migration run and
// is not pool-managed, so it gets its own connection here rather than holding a
// pooled one. Startup-only; callers close it when the migration run completes
// (or leave it for the process, as the migrator currently does).
func MigrationConn(ctx context.Context) (*pgx.Conn, error) {
	conn, err := connectOne(ctx)
	if err == nil {
		return conn, nil
	}
	if !isAuthError(err) || config.GetInstance().Db.SecretId == "" {
		return nil, err
	}
	log.Println("db authentication failed; refreshing credentials and retrying once")
	if rerr := refreshDbCreds(ctx); rerr != nil {
		log.Println("could not refresh db credentials", rerr)
		return nil, err
	}
	return connectOne(ctx)
}

// connectOne builds the connection config from the current credentials and
// opens a single (non-pooled) connection. Used by the migrator.
func connectOne(ctx context.Context) (*pgx.Conn, error) {
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
// the next connection picks up the post-rotation password. Caches the secret on
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

	// otherwise pull user/pass from the secret. Call once.Do unconditionally (no
	// bare dbSecret nil-check first): once.Do is what establishes the happens-
	// before for the dbSecret write, so reading the pointer outside it - from
	// concurrent request goroutines at startup - would be an unsynchronized read.
	// err is set only on the goroutine that ran the init, so re-check dbSecret
	// afterward to surface a failed init to every caller.
	var err error
	once.Do(func() {
		dbSecret, err = secrets.NewSecret(cfg.Db.SecretId)
	})
	if err != nil {
		return nil, err
	}
	if dbSecret == nil {
		return nil, errors.New("db secret initialization failed")
	}

	creds := &dbCreds{}
	err = dbSecret.Unmarshal(creds)
	if err != nil {
		log.Println("could not unmarshal credentials", err)
		return nil, err
	}

	return creds, nil
}

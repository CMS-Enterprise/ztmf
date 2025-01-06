package db

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
	"github.com/jackc/pgx/v5"
)

var (
	once     sync.Once
	dbSecret *secrets.Secret
)

type dbCreds struct {
	Username string
	Password string
}

func Conn(ctx context.Context) (*pgx.Conn, error) {
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

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		log.Println("could not connect to db", err)
		return nil, err
	}

	return conn, err
}

func getDbCreds() (*dbCreds, error) {
	cfg := config.GetInstance()

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

	secVal, err := dbSecret.Value()
	if err != nil {
		return nil, err
	}
	creds := &dbCreds{}
	err = json.Unmarshal([]byte(*secVal), creds)
	if err != nil {
		log.Println("could not unmarshal credentials", err)
		return nil, err
	}

	return creds, nil
}

package db

import (
	"context"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var once sync.Once
var dbpool *pgxpool.Pool

func GetPool() *pgxpool.Pool {

	if dbpool == nil {
		once.Do(func() {

			cfg := config.GetConfig()
			var err error
			// dbpool, err := pgxpool.New(context.Background(), "postgres://"+cfg.DB.User+":"+cfg.DB.Pass+"@"+cfg.DB.Endpoint+":"+cfg.DB.Port+"/"+cfg.DB.Database+"?sslmode=verify-ca&pool_max_conns=10")
			dbpool, err = pgxpool.New(context.Background(), "user="+cfg.DB.User+" password="+cfg.DB.Pass+" host="+cfg.DB.Endpoint+" port="+cfg.DB.Port+" dbname="+cfg.DB.Database+" sslmode=allow pool_max_conns=10")
			if err != nil {
				log.Fatal(err)
			}
		})
	}
	return dbpool
}

package engine

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getDb() (*pgxpool.Pool, error) {
	cfg = GetConfig()

	// dbpool, err := pgxpool.New(context.Background(), "postgres://"+cfg.DB.User+":"+cfg.DB.Pass+"@"+cfg.DB.Endpoint+":"+cfg.DB.Port+"/"+cfg.DB.Database+"?sslmode=verify-ca&pool_max_conns=10")
	dbpool, err := pgxpool.New(context.Background(), "user="+cfg.DB.User+" password="+cfg.DB.Pass+" host="+cfg.DB.Endpoint+" port="+cfg.DB.Port+" dbname="+cfg.DB.Database+" sslmode=allow pool_max_conns=10")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return dbpool, err
}

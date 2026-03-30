package database

import (
	"context"
	"history-api/pkg/config"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresqlDB() (*pgxpool.Pool, error) {
	ctx := context.Background()

	connectionURI, err := config.GetConfig("PGX_CONNECTION_URI")
	if err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(connectionURI)
	if err != nil {
		return nil, err
	}

	var pool *pgxpool.Pool

	for i := 0; i < 5; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if err = pool.Ping(ctx); err == nil {
			return pool, nil
		}

		time.Sleep(2 * time.Second)
	}

	return nil, err
}

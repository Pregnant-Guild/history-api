package database

import (
	"context"
	"history-api/pkg/config"

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

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}
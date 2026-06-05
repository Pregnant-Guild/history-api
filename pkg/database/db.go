package database

import (
	"context"
	"history-api/pkg/config"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func defaultMaxConns() int {
	conns := runtime.NumCPU() * 8
	if conns < 16 {
		return 16
	}
	if conns > 64 {
		return 64
	}
	return conns
}

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
	maxConns := config.GetIntConfigWithDefault("PGX_MAX_CONNS", defaultMaxConns())
	if maxConns < 1 {
		maxConns = defaultMaxConns()
	}

	minConns := config.GetIntConfigWithDefault("PGX_MIN_CONNS", maxConns/4)
	if minConns < 0 {
		minConns = 0
	}
	if minConns > maxConns {
		minConns = maxConns
	}

	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = int32(minConns)
	poolConfig.MaxConnIdleTime = time.Duration(config.GetIntConfigWithDefault("PGX_MAX_CONN_IDLE_SECONDS", 300)) * time.Second
	poolConfig.HealthCheckPeriod = time.Duration(config.GetIntConfigWithDefault("PGX_HEALTH_CHECK_SECONDS", 60)) * time.Second

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

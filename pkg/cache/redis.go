package cache

import (
	"context"
	"fmt"
	"history-api/pkg/config"

	"github.com/redis/go-redis/v9"
)

var RI *redis.Client

func Connect() error {
	connectionURI, err := config.GetConfig("REDIS_CONNECTION_URI")

	if err != nil {
		return err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     connectionURI,
		Password: "",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("Could not connect to Redis: %v", err)
	}

	RI = rdb
	return nil
}

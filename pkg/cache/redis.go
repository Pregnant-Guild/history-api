package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) error
	Del(ctx context.Context, keys ...string) error
	DelByPattern(ctx context.Context, pattern string) error
	MGet(ctx context.Context, keys ...string) [][]byte
	MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
	GetRawClient() *redis.Client
	PublishTask(ctx context.Context, streamName string, taskType constants.TaskType, payload any) error
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient() (Cache, error) {
	uri, err := config.GetConfig("REDIS_CONNECTION_URI")
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         uri,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		DisableIdentity: true,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to Redis: %v", err)
	}
	return &RedisClient{client: rdb}, nil
}

func (r *RedisClient) GetRawClient() *redis.Client {
	return r.client
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) DelByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return fmt.Errorf("error scanning keys with pattern %s: %v", pattern, err)
		}

		if len(keys) > 0 {
			if err := r.client.Unlink(ctx, keys...).Err(); err != nil {
                return fmt.Errorf("error unlinking keys during scan: %v", err)
            }
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string, dest any) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (r *RedisClient) MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error {
	pipe := r.client.Pipeline()
	for key, value := range pairs {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal key %s: %v", key, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisClient) MGet(ctx context.Context, keys ...string) [][]byte {
	res, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil
	}
	results := make([][]byte, len(res))
	for i, val := range res {
		if val != nil {
			results[i] = []byte(val.(string))
		}
	}
	return results
}

func (r *RedisClient) PublishTask(ctx context.Context, streamName string, taskType constants.TaskType, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"task_type": taskType.String(),
			"payload":   string(payloadBytes),
		},
	}).Err()
}

func GetMultiple[T any](ctx context.Context, c Cache, keys []string) ([]T, error) {
	raws := c.MGet(ctx, keys...)
	final := make([]T, 0)
	for _, b := range raws {
		if b == nil {
			continue
		}
		var item T
		if err := json.Unmarshal(b, &item); err == nil {
			final = append(final, item)
		}
	}
	return final, nil
}

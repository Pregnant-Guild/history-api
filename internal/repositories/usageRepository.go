package repositories

import (
	"context"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"time"
)

type UsageRepository interface {
	GetAIUsage(ctx context.Context, userID string) (int, error)
	IncrementAIUsage(ctx context.Context, userID string) (int, error)
}

type usageRepository struct {
	c cache.Cache
}

func NewUsageRepository(c cache.Cache) UsageRepository {
	return &usageRepository{
		c: c,
	}
}

func (r *usageRepository) getUsageKey(userID string) string {
	dateStr := time.Now().Format("20060102")
	return cache.Key2("usage:ai", userID, dateStr)
}

func (r *usageRepository) GetAIUsage(ctx context.Context, userID string) (int, error) {
	key := r.getUsageKey(userID)
	var count int
	err := r.c.Get(ctx, key, &count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

func (r *usageRepository) IncrementAIUsage(ctx context.Context, userID string) (int, error) {
	key := r.getUsageKey(userID)
	rdb := r.c.GetRawClient()

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		rdb.Expire(ctx, key, constants.UsageExpiration)
	}

	return int(count), nil
}

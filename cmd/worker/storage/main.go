package main

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	_ "history-api/pkg/log"
	"history-api/pkg/storage"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func runSingleWorker(ctx context.Context, rdb *redis.Client, consumerID int, sc storage.Storage) {
	consumerName := "worker-" + strconv.Itoa(consumerID)

	log.Info().Str("worker", consumerName).Msg("Worker started and ready")

	for {
		entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    constants.GroupStorageName,
			Consumer: consumerName,
			Streams:  []string{constants.StreamStorageName, ">"},
			Count:    1,
			Block:    0,
		}).Result()

		if err != nil {
			log.Error().Err(err).Str("worker", consumerName).Msg("Failed to read stream")
			time.Sleep(2 * time.Second)
			continue
		}

		for _, stream := range entries {
			for _, message := range stream.Messages {
				taskType, ok1 := message.Values["task_type"].(string)
				payloadStr, ok2 := message.Values["payload"].(string)
				if !ok1 || !ok2 {
					log.Error().Msg("Invalid message format")
					rdb.XAck(ctx, constants.StreamStorageName, constants.GroupStorageName, message.ID)
					continue
				}

				if taskType == constants.TaskTypeDeleteMedia.String() {
					var data models.MediaStorageEntity
					if err := json.Unmarshal([]byte(payloadStr), &data); err != nil {
						log.Error().Err(err).Msg("Failed to unmarshal payload")
						continue
					}

					log.Info().
						Str("worker", consumerName).
						Str("storage_key", data.StorageKey).
						Msg("Processing delete media task")

					errSend := sc.Delete(ctx, data.StorageKey)
					if errSend != nil {
						log.Error().Err(errSend).Str("storage_key", data.StorageKey).Msg("Failed to delete media")
						continue
					}
				}

				if taskType == constants.TaskTypeBulkDeleteMedia.String() {
					var data []*models.MediaStorageEntity
					if err := json.Unmarshal([]byte(payloadStr), &data); err != nil {
						log.Error().Err(err).Msg("Failed to unmarshal payload")
						continue
					}
					storageKeys := make([]string, len(data))
					for i, item := range data {
						storageKeys[i] = item.StorageKey
					}
					log.Info().
						Str("worker", consumerName).
						Int("count", len(storageKeys)).
						Msg("Processing bulk delete media task")
					errSend := sc.BulkDelete(ctx, storageKeys)
					if errSend != nil {
						log.Error().Err(errSend).Msg("Failed to bulk delete")
						continue
					}
				}

				rdb.XAck(ctx, constants.StreamStorageName, constants.GroupStorageName, message.ID)
				log.Info().Str("msg_id", message.ID).Msg("Task acknowledged")
			}
		}
	}
}
func main() {

	config.LoadEnv()

	workerCountStr := config.GetConfigWithDefault("STORAGE_WORKER_COUNT", "1")
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil || workerCount <= 0 {
		workerCount = 1
	}

	cacheInterface, err := cache.NewRedisClient()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to connect to Redis")
	}

	rdb := cacheInterface.GetRawClient()

	sc, err := storage.NewS3Storage()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to create S3 storage client")
	}

	ctx := context.Background()

	err = rdb.XGroupCreateMkStream(ctx, constants.StreamStorageName, constants.GroupStorageName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Fatal().
			Err(err).
			Msg("Failed to create Redis Stream Group")
	}

	log.Info().
		Int("worker_count", workerCount).
		Msg("Starting storage worker system")

	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Go(func() {
			runSingleWorker(ctx, rdb, i, sc)
		})
	}

	wg.Wait()
}

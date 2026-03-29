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
	"history-api/pkg/email"
	_ "history-api/pkg/log"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func runSingleWorker(ctx context.Context, rdb *redis.Client, consumerID int) {
	consumerName := "worker-" + strconv.Itoa(consumerID)

	log.Info().Str("worker", consumerName).Msg("Worker started and ready")

	for {
		entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    constants.GroupEmailName,
			Consumer: consumerName,
			Streams:  []string{constants.StreamEmailName, ">"},
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
				taskType := message.Values["task_type"].(string)
				payloadStr := message.Values["payload"].(string)

				if taskType == constants.TaskTypeSendEmailOTP.String() {
					var data models.TokenEntity
					if err := json.Unmarshal([]byte(payloadStr), &data); err != nil {
						log.Error().Err(err).Msg("Failed to unmarshal payload")
						continue
					}

					log.Info().
						Str("worker", consumerName).
						Str("email", data.Email).
						Msg("Processing email task")

					errSend := email.SendMailOTP(data.Email, data.Token, data.TokenType)
					if errSend != nil {
						log.Error().Err(errSend).Str("email", data.Email).Msg("Failed to send email")
						continue
					}
				}

				rdb.XAck(ctx, constants.StreamEmailName, constants.GroupEmailName, message.ID)
				log.Info().Str("msg_id", message.ID).Msg("Task acknowledged")
			}
		}
	}
}
func main() {

	config.LoadEnv()

	workerCountStr := config.GetConfigWithDefault("EMAIL_WORKER_COUNT", "1")
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
	ctx := context.Background()

	err = rdb.XGroupCreateMkStream(ctx, constants.StreamEmailName, constants.GroupEmailName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Fatal().
			Err(err).
			Msg("Failed to create Redis Stream Group")
	}

	log.Info().
		Int("worker_count", workerCount).
		Msg("Starting email worker system")

	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Go(func() {
			runSingleWorker(ctx, rdb, i)
		})
	}

	wg.Wait()
}

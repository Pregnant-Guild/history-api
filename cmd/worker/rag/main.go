package main

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"time"

	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/ai"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"history-api/pkg/database"
	_ "history-api/pkg/log"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	maxRetries     = 3
	baseRetryDelay = 2 * time.Second
	itemDelay      = 500 * time.Millisecond
)

func processRagTask(ctx context.Context, ragRepo repositories.RagRepository, ragUtils *ai.RagUtils, task *models.RagIndexTask, workerName string) {
	if len(task.DeleteWikiIDs) > 0 {
		if err := ragRepo.DeleteBySourceIDs(ctx, "wiki", task.DeleteWikiIDs); err != nil {
			log.Error().Err(err).Str("worker", workerName).Msg("Failed to delete wiki RAG chunks")
		}
	}

	if len(task.DeleteEntityIDs) > 0 {
		if err := ragRepo.DeleteBySourceIDs(ctx, "entity", task.DeleteEntityIDs); err != nil {
			log.Error().Err(err).Str("worker", workerName).Msg("Failed to delete entity RAG chunks")
		}
	}

	// 2. Index wikis with delay + retry
	for _, wiki := range task.Wikis {
		if wiki.Source != "inline" {
			continue
		}

		log.Info().Str("worker", workerName).Str("wiki_id", wiki.ID).Msg("Indexing wiki")

		cleanText := ragUtils.StripHTML(wiki.Title + "\n" + wiki.Doc)

		var chunks []string
		var vectors [][]float32
		var err error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				delay := baseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
				log.Warn().
					Str("worker", workerName).
					Str("wiki_id", wiki.ID).
					Int("attempt", attempt).
					Dur("delay", delay).
					Msg("Retrying wiki embedding")
				time.Sleep(delay)
			}

			chunks, vectors, err = ragUtils.PrepareChunks(ctx, cleanText)
			if err == nil {
				break
			}

			log.Error().Err(err).
				Str("worker", workerName).
				Str("wiki_id", wiki.ID).
				Int("attempt", attempt).
				Msg("Failed to prepare wiki chunks")
		}

		if err != nil {
			log.Error().Err(err).Str("worker", workerName).Str("wiki_id", wiki.ID).Msg("Giving up on wiki after max retries")
			continue
		}

		// Delete existing chunks then save new ones
		_ = ragRepo.DeleteBySourceIDs(ctx, "wiki", []string{wiki.ID})
		for i, chunk := range chunks {
			if saveErr := ragRepo.SaveChunk(ctx, "wiki", wiki.ID, task.ProjectID, i, chunk, vectors[i]); saveErr != nil {
				log.Error().Err(saveErr).Str("wiki_id", wiki.ID).Int("chunk", i).Msg("Failed to save wiki chunk")
			}
		}

		log.Info().Str("worker", workerName).Str("wiki_id", wiki.ID).Int("chunks", len(chunks)).Msg("Wiki indexed successfully")

		// Delay between items to avoid rate limit
		time.Sleep(itemDelay)
	}

	// 3. Index entities with delay + retry
	for _, entity := range task.Entities {
		if entity.Source != "inline" {
			continue
		}

		log.Info().Str("worker", workerName).Str("entity_id", entity.ID).Msg("Indexing entity")

		cleanText := ragUtils.StripHTML(entity.Name + "\n" + entity.Description)

		var chunks []string
		var vectors [][]float32
		var err error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				delay := baseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
				log.Warn().
					Str("worker", workerName).
					Str("entity_id", entity.ID).
					Int("attempt", attempt).
					Dur("delay", delay).
					Msg("Retrying entity embedding")
				time.Sleep(delay)
			}

			chunks, vectors, err = ragUtils.PrepareChunks(ctx, cleanText)
			if err == nil {
				break
			}

			log.Error().Err(err).
				Str("worker", workerName).
				Str("entity_id", entity.ID).
				Int("attempt", attempt).
				Msg("Failed to prepare entity chunks")
		}

		if err != nil {
			log.Error().Err(err).Str("worker", workerName).Str("entity_id", entity.ID).Msg("Giving up on entity after max retries")
			continue
		}

		// Delete existing chunks then save new ones
		_ = ragRepo.DeleteBySourceIDs(ctx, "entity", []string{entity.ID})
		for i, chunk := range chunks {
			if saveErr := ragRepo.SaveChunk(ctx, "entity", entity.ID, task.ProjectID, i, chunk, vectors[i]); saveErr != nil {
				log.Error().Err(saveErr).Str("entity_id", entity.ID).Int("chunk", i).Msg("Failed to save entity chunk")
			}
		}

		log.Info().Str("worker", workerName).Str("entity_id", entity.ID).Int("chunks", len(chunks)).Msg("Entity indexed successfully")
		time.Sleep(itemDelay)
	}
}

func runSingleWorker(ctx context.Context, rdb *redis.Client, consumerID int, ragRepo repositories.RagRepository, ragUtils *ai.RagUtils) {
	consumerName := "worker-" + strconv.Itoa(consumerID)

	log.Info().Str("worker", consumerName).Msg("RAG worker started and ready")

	for {
		entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    constants.GroupRagName,
			Consumer: consumerName,
			Streams:  []string{constants.StreamRagName, ">"},
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
					rdb.XAck(ctx, constants.StreamRagName, constants.GroupRagName, message.ID)
					continue
				}

				if taskType == constants.TaskTypeRagIndexSubmission.String() {
					var task models.RagIndexTask
					if err := json.Unmarshal([]byte(payloadStr), &task); err != nil {
						log.Error().Err(err).Msg("Failed to unmarshal RAG task payload")
						rdb.XAck(ctx, constants.StreamRagName, constants.GroupRagName, message.ID)
						continue
					}

					log.Info().
						Str("worker", consumerName).
						Str("project_id", task.ProjectID).
						Int("wikis", len(task.Wikis)).
						Int("entities", len(task.Entities)).
						Msg("Processing RAG index task")

					processRagTask(ctx, ragRepo, ragUtils, &task, consumerName)
				}

				rdb.XAck(ctx, constants.StreamRagName, constants.GroupRagName, message.ID)
				log.Info().Str("msg_id", message.ID).Msg("Task acknowledged")
			}
		}
	}
}

func main() {
	config.LoadEnv()

	workerCountStr := config.GetConfigWithDefault("RAG_WORKER_COUNT", "1")
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

	poolPg, err := database.NewPostgresqlDB()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to connect to PostgreSQL")
	}
	defer poolPg.Close()

	ragUtils, err := ai.NewRagUtils()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to initialize RAG utils")
	}

	ragRepo := repositories.NewRagRepository(poolPg, cacheInterface)

	ctx := context.Background()

	err = rdb.XGroupCreateMkStream(ctx, constants.StreamRagName, constants.GroupRagName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Fatal().
			Err(err).
			Msg("Failed to create Redis Stream Group")
	}

	log.Info().
		Int("worker_count", workerCount).
		Msg("Starting RAG worker system")

	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Go(func() {
			runSingleWorker(ctx, rdb, i, ragRepo, ragUtils)
		})
	}

	wg.Wait()
}

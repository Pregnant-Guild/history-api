package main

import (
	"context"
	"fmt"
	"history-api/internal/repositories"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/storage"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func runStatistics(ctx context.Context, repo repositories.StatisticRepository) {
	log.Info().Msg("Running daily statistics...")
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := repo.Upsert(ctx, today)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upsert system statistics")
	} else {
		log.Info().Msg("Successfully updated daily statistics and cleared cache")
	}
}

func runBackup(ctx context.Context, s3 storage.Storage, dbURI string) {
	log.Info().Msg("Running weekly database backup...")

	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("db_backup_%s.sql", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(tmpDir, fileName)

	// Run pg_dump
	cmd := exec.Command("pg_dump", dbURI, "-F", "c", "-f", filePath)
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("Failed to execute pg_dump. Make sure pg_dump is installed.")
		return
	}
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open backup file")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Error().Err(err).Msg("Failed to stat backup file")
		return
	}

	key := fmt.Sprintf("backups/%s", fileName)
	err = s3.Upload(ctx, key, file, stat.Size(), storage.UploadOptions{
		ContentType: "application/octet-stream",
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to upload backup to S3")
	} else {
		log.Info().Str("key", key).Msg("Successfully uploaded backup to S3")
	}
}

func main() {
	config.LoadEnv()

	dbURI, err := config.GetConfig("DATABASE_URI")
	if err != nil {
		log.Fatal().Err(err).Msg("DATABASE_URI not set")
	}

	dbPool, err := pgxpool.New(context.Background(), dbURI)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	defer dbPool.Close()

	redis, err := cache.NewRedisClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	statisticRepo := repositories.NewStatisticRepository(dbPool, redis)

	s3Store, err := storage.NewS3Storage()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init S3 storage")
	}

	log.Info().Msg("Cron worker started")

	// Run initially on startup
	runStatistics(context.Background(), statisticRepo)

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create scheduler")
	}

	// Run statistics every day at midnight (00:00)
	_, err = s.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))),
		gocron.NewTask(func() {
			runStatistics(context.Background(), statisticRepo)
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to schedule daily statistics")
	}

	// Run backup every Sunday at 01:00 AM
	_, err = s.NewJob(
		gocron.WeeklyJob(1, gocron.NewWeekdays(time.Sunday), gocron.NewAtTimes(gocron.NewAtTime(1, 0, 0))),
		gocron.NewTask(func() {
			runBackup(context.Background(), s3Store, dbURI)
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to schedule weekly backup")
	}

	s.Start()

	select {}
}

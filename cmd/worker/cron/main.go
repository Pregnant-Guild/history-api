package main

import (
	"context"
	"fmt"
	"history-api/internal/repositories"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/database"
	"history-api/pkg/storage"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

func runStatistics(ctx context.Context, repo repositories.StatisticRepository) {
	log.Info().Msg("Running daily statistics...")

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load Asia/Ho_Chi_Minh timezone, falling back to fixed UTC+7")
		loc = time.FixedZone("ICT", 7*3600)
	}

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Upsert stats for today, yesterday, and the day before to prevent timezone gaps/delays
	for i := 0; i < 3; i++ {
		date := today.AddDate(0, 0, -i)
		log.Info().Str("date", date.Format("2006-01-02")).Msg("Upserting system statistics")
		_, err = repo.Upsert(ctx, date)
		if err != nil {
			log.Error().Err(err).Str("date", date.Format("2006-01-02")).Msg("Failed to upsert system statistics")
		}
	}
	log.Info().Msg("Successfully updated daily statistics and cleared cache")
}

func runBackup(ctx context.Context, s3 storage.Storage, dbURI string) {
	log.Info().Msg("Running weekly database backup...")

	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("db_backup_%s.sql", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(tmpDir, fileName)

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
	err := config.LoadEnv()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}
	connectionURI, err := config.GetConfig("PGX_CONNECTION_URI")
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	poolPg, err := database.NewPostgresqlDB()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}
	defer poolPg.Close()

	redis, err := cache.NewRedisClient()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	statisticRepo := repositories.NewStatisticRepository(poolPg, redis)

	s3Store, err := storage.NewS3Storage()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	log.Info().Msg("Cron worker started")

	// Run initially on startup
	runStatistics(context.Background(), statisticRepo)

	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create scheduler")
	}

	// Run statistics every day at 01:00 AM
	_, err = s.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(1, 0, 0))),
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
			runBackup(context.Background(), s3Store, connectionURI)
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to schedule weekly backup")
	}

	s.Start()

	select {}
}

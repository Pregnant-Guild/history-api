package main

import (
	"context"
	"fmt"
	_ "history-api/docs"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/database"
	_ "history-api/pkg/log"
	"history-api/pkg/mbtiles"
	"history-api/pkg/oauth"
	"history-api/pkg/storage"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func gracefulShutdown(fiberServer *FiberServer, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Info().Msg("shutting down gracefully, press Ctrl+C again to force")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := fiberServer.App.ShutdownWithContext(ctx); err != nil {
		log.Info().Msgf("Server forced to shutdown with error: %v", err)
	}

	log.Info().Msg("Server exiting")

	done <- true
}

//export StartServer
func StartServer() {
	err := config.LoadEnv()
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

	err = database.SeedSuperAdmin(poolPg)
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	sqlTile, err := mbtiles.NewMBTilesDB("data/map.mbtiles")
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}
	defer sqlTile.Close()

	redisClient, err := cache.NewRedisClient()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	storageClient, err := storage.NewS3Storage()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	googleOAuthConfig, err := oauth.NewGoogleProvider()
	if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}

	serverIp, _ := config.GetConfig("SERVER_IP")
	if serverIp == "" {
		serverIp = "127.0.0.1"
	}

	httpPort, _ := config.GetConfig("SERVER_PORT")
	if httpPort == "" {
		httpPort = "3434"
	}

	serverHttp := NewHttpServer()
	serverHttp.SetupServer(poolPg, sqlTile, redisClient, storageClient, googleOAuthConfig)
	Singleton = serverHttp

	done := make(chan bool, 1)

	err = serverHttp.App.Listen(fmt.Sprintf("%s:%s", serverIp, httpPort))
	if err != nil {
		log.Error().Msgf("Error: app failed to start on port %s, %v", httpPort, err)
		panic(err)
	}

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(serverHttp, done)

	// Wait for the graceful shutdown to complete
	<-done
	log.Info().Msg("Graceful shutdown complete.")
}

// @title           History API
// @version         1.0
// @description     This is a sample server for History API.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer " followed by a space and JWT token.
func main() {
	StartServer()
}

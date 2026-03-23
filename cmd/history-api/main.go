package main

import (
	"context"
	// _ "history-api/docs"
	"history-api/internal/gen/sqlc"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	_ "history-api/pkg/log"

	"fmt"
	"history-api/pkg/database"
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
	pool, err := database.Connect()
		if err != nil {
		log.Error().Msg(err.Error())
		panic(err)
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	err = cache.Connect()
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
	serverHttp.RegisterFiberRoutes()
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

// @title Firefly Manager API
// @version 1.0
// @description API to update Firefly Manager data
// @host localhost:3344
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	StartServer()
}

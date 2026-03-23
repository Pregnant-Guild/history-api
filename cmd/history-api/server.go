package main

import (
	// "history-api/internal/routes"
	// "history-api/internal/services"

	swagger "github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

var (
	Singleton *FiberServer
)

type FiberServer struct {
	App *fiber.App
}

func NewHttpServer() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "http-server",
			AppName:      "http-server",
		}),
	}
	cfg := swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "swagger",
		Title:    "Swagger API Docs",
	}

	server.App.Use(swagger.New(cfg))
	server.App.Use(logger.New())
	return server
}

func (s *FiberServer) RegisterFiberRoutes() {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// routes.UserRoutes(s.App)
	// routes.AuthRoutes(s.App)
	// routes.MediaRoute(s.App)
	// routes.NotFoundRoute(s.App)

}

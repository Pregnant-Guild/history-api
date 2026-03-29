package main

import (
	"database/sql"
	_ "embed"
	"history-api/docs"
	"history-api/internal/controllers"
	"history-api/internal/gen/sqlc"
	"history-api/internal/repositories"
	"history-api/internal/routes"
	"history-api/internal/services"
	"history-api/pkg/cache"
	"os"

	swagger "github.com/gofiber/contrib/v3/swaggerui"
	middleware "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/rs/zerolog"
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
		BasePath:    "/",
		FileContent: docs.SwaggerJSON,
		Path:        "swagger",
		Title:       "Swagger API Docs",
	}

	server.App.Use(swagger.New(cfg))

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	server.App.Use(middleware.New(middleware.Config{
		Logger: &logger,
	}))
	return server
}

func (s *FiberServer) SetupServer(sqlPg sqlc.DBTX, sqlTile *sql.DB, redis cache.Cache) {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// repo setup
	userRepo := repositories.NewUserRepository(sqlPg, redis)
	roleRepo := repositories.NewRoleRepository(sqlPg, redis)
	tileRepo := repositories.NewTileRepository(sqlTile, redis)
	tokenRepo := repositories.NewTokenRepository(redis)

	// service setup
	authService := services.NewAuthService(userRepo, roleRepo, tokenRepo, redis)
	userService := services.NewUserService(userRepo, roleRepo)
	tileService := services.NewTileService(tileRepo)

	// controller setup
	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(userService)
	tileController := controllers.NewTileController(tileService)

	// route setup
	routes.AuthRoutes(s.App, authController, userRepo)
	routes.UserRoutes(s.App, userController, userRepo)
	routes.TileRoutes(s.App, tileController)
	routes.NotFoundRoute(s.App)
}

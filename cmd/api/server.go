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
	"history-api/pkg/storage"
	"os"
	"time"

	swagger "github.com/gofiber/contrib/v3/swaggerui"
	middleware "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
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

	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()
	server.App.Use(middleware.New(middleware.Config{
		Logger: &logger,
	}))
	return server
}

func (s *FiberServer) SetupServer(sqlPg sqlc.DBTX, sqlTile *sql.DB, redis cache.Cache, sclient storage.Storage, oauth *oauth2.Config) {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://localhost:3000",
			"http://localhost:3001",
			"https://localhost:3001",
			"http://localhost:3344",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin"},
		AllowCredentials: true,
	}))

	// repo setup
	userRepo := repositories.NewUserRepository(sqlPg, redis)
	roleRepo := repositories.NewRoleRepository(sqlPg, redis)
	tileRepo := repositories.NewTileRepository(sqlTile, redis)
	tokenRepo := repositories.NewTokenRepository(redis)
	mediaRepo := repositories.NewMediaRepository(sqlPg, redis)

	// service setup
	authService := services.NewAuthService(userRepo, roleRepo, tokenRepo, redis)
	userService := services.NewUserService(userRepo, roleRepo, redis)
	roleService := services.NewRoleService(roleRepo)
	tileService := services.NewTileService(tileRepo)
	mediaService := services.NewMediaService(mediaRepo, tokenRepo, sclient, redis)

	// controller setup
	authController := controllers.NewAuthController(authService, oauth)
	userController := controllers.NewUserController(userService, mediaService)
	tileController := controllers.NewTileController(tileService)
	roleController := controllers.NewRoleController(roleService)
	mediaController := controllers.NewMediaController(mediaService)

	// route setup
	routes.AuthRoutes(s.App, authController, userRepo)
	routes.UserRoutes(s.App, userController, userRepo)
	routes.MediaRoutes(s.App, mediaController, userRepo)
	routes.RoleRoutes(s.App, roleController, userRepo)
	routes.TileRoutes(s.App, tileController)
	routes.NotFoundRoute(s.App)
}

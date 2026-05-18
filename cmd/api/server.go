package main

import (
	"database/sql"
	_ "embed"
	"history-api/docs"
	"history-api/internal/controllers"
	"history-api/internal/repositories"
	"history-api/internal/routes"
	"history-api/internal/services"
	"history-api/pkg/ai"
	"history-api/pkg/cache"
	"history-api/pkg/storage"
	"os"
	"time"

	swagger "github.com/gofiber/contrib/v3/swaggerui"
	middleware "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *FiberServer) SetupServer(
	poolPg *pgxpool.Pool,
	sqlTile *sql.DB,
	sqlRasterTile *sql.DB,
	redis cache.Cache,
	sclient storage.Storage,
	oauth *oauth2.Config,
	raguUtils *ai.RagUtils,
) {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"https://history-admin.kain.id.vn",
			"https://history-user.kain.id.vn",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Origin"},
		AllowCredentials: true,
	}))

	// repo setup
	userRepo := repositories.NewUserRepository(poolPg, redis)
	roleRepo := repositories.NewRoleRepository(poolPg, redis)
	tileRepo := repositories.NewTileRepository(sqlTile, redis)
	rasterTileRepo := repositories.NewRasterTileRepository(sqlRasterTile, redis)
	tokenRepo := repositories.NewTokenRepository(redis)
	mediaRepo := repositories.NewMediaRepository(poolPg, redis)
	verificationRepo := repositories.NewVerificationRepository(poolPg, redis)
	entityRepo := repositories.NewEntityRepository(poolPg, redis)
	geometryRepo := repositories.NewGeometryRepository(poolPg, redis)
	wikiRepo := repositories.NewWikiRepository(poolPg, redis)
	projectRepo := repositories.NewProjectRepository(poolPg, redis)
	commitRepo := repositories.NewCommitRepository(poolPg, redis)
	submissionRepo := repositories.NewSubmissionRepository(poolPg, redis)

	raguRepo := repositories.NewRagRepository(poolPg, redis)
	usageRepo := repositories.NewUsageRepository(redis)
	statisticRepo := repositories.NewStatisticRepository(poolPg, redis)
	chatRepo := repositories.NewChatRepository(poolPg, redis)
	battleReplayRepo := repositories.NewBattleReplayRepository(poolPg, redis)

	// service setup
	authService := services.NewAuthService(userRepo, roleRepo, tokenRepo, redis, poolPg)
	userService := services.NewUserService(userRepo, roleRepo, redis, poolPg)
	roleService := services.NewRoleService(roleRepo)
	tileService := services.NewTileService(tileRepo)
	rasterTileService := services.NewRasterTileService(rasterTileRepo)
	mediaService := services.NewMediaService(mediaRepo, tokenRepo, sclient, redis)
	verificationService := services.NewVerificationService(verificationRepo, mediaRepo, userRepo, roleRepo, redis, poolPg)
	entityService := services.NewEntityService(entityRepo)
	geometryService := services.NewGeometryService(geometryRepo)
	wikiService := services.NewWikiService(wikiRepo)
	projectService := services.NewProjectService(projectRepo)
	commitService := services.NewCommitService(poolPg, commitRepo, projectRepo, redis)
	submissionService := services.NewSubmissionService(
		submissionRepo, projectRepo, commitRepo,
		userRepo, wikiRepo, geometryRepo, entityRepo,
		battleReplayRepo, raguRepo, raguUtils, poolPg, redis,
	)
	chatbotService := services.NewChatbotService(raguRepo, usageRepo, chatRepo, raguUtils)
	statisticService := services.NewStatisticService(statisticRepo)
	battleReplayService := services.NewBattleReplayService(battleReplayRepo)
	goongService := services.NewGoongService(redis)

	// controller setup
	authController := controllers.NewAuthController(authService, oauth)
	userController := controllers.NewUserController(userService, mediaService, verificationService, projectService)
	tileController := controllers.NewTileController(tileService)
	rasterTileController := controllers.NewRasterTileController(rasterTileService)
	roleController := controllers.NewRoleController(roleService)
	mediaController := controllers.NewMediaController(mediaService)
	verificationController := controllers.NewVerificationController(verificationService)
	entityController := controllers.NewEntityController(entityService)
	geometryController := controllers.NewGeometryController(geometryService)
	wikiController := controllers.NewWikiController(wikiService)
	projectController := controllers.NewProjectController(projectService)
	commitController := controllers.NewCommitController(commitService)
	submissionController := controllers.NewSubmissionController(submissionService)
	chatbotController := controllers.NewChatbotController(chatbotService)
	statisticController := controllers.NewStatisticController(statisticService)
	battleReplayController := controllers.NewBattleReplayController(battleReplayService)
	goongController := controllers.NewGoongController(goongService)

	// route setup
	routes.AuthRoutes(s.App, authController, userRepo)
	routes.UserRoutes(s.App, userController, userRepo)
	routes.MediaRoutes(s.App, mediaController, userRepo)
	routes.RoleRoutes(s.App, roleController, userRepo)
	routes.VerificationRoutes(s.App, verificationController, userRepo)
	routes.TileRoutes(s.App, tileController)
	routes.RasterTileRoutes(s.App, rasterTileController)
	routes.EntityRoutes(s.App, entityController)
	routes.GeometryRoutes(s.App, geometryController)
	routes.WikiRoutes(s.App, wikiController)
	routes.ProjectRoutes(s.App, projectController, commitController, userRepo)
	routes.SubmissionRoutes(s.App, submissionController, userRepo)
	routes.ChatbotRoutes(s.App, chatbotController, userRepo)
	routes.StatisticRoutes(s.App, statisticController, userRepo)
	routes.BattleReplayRoutes(s.App, battleReplayController)
	routes.GoongRoutes(s.App, goongController)
	routes.NotFoundRoute(s.App)
}

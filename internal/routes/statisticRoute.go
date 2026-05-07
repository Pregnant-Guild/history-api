package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func StatisticRoutes(app *fiber.App, statController *controllers.StatisticController, userRepo repositories.UserRepository) {
	statGroup := app.Group(
		"/statistics",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
	)

	statGroup.Get("/", statController.SearchStatistics)
	statGroup.Get("/:date", statController.GetStatisticByDate)
}


package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func RoleRoutes(app *fiber.App, controller *controllers.RoleController, userRepo repositories.UserRepository) {
	route := app.Group("/roles")

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.GetRoleById,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		controller.GetAllRole,
	)
}

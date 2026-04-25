package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func ProjectRoutes(app *fiber.App, controller *controllers.ProjectController, userRepo repositories.UserRepository) {
	route := app.Group("/projects")

	route.Get(
		"/:id",
		controller.GetProjectByID,
	)

	route.Put(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.UpdateProject,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.DeleteProject,
	)

	route.Get(
		"/",
		controller.SearchProject,
	)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		controller.CreateProject,
	)
}

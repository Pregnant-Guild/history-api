package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func SubmissionRoutes(
	app *fiber.App,
	controller controllers.SubmissionController,
	userRepo repositories.UserRepository,
) {
	route := app.Group("/submissions")

	route.Patch(
		"/:id/status",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.UpdateSubmissionStatus,
	)

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.GetSubmissionByID,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.DeleteSubmission,
	)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		controller.CreateSubmission,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.SearchSubmissions,
	)

}

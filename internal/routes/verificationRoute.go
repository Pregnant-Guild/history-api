package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func VerificationRoutes(app *fiber.App, controller *controllers.VerificationController, userRepo repositories.UserRepository) {
	route := app.Group("/historian/application")

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetVerificationByID,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.DeleteVerification,
	)

	route.Put(
		"/:id/status",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.UpdateVerificationStatus,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.SearchVerification,
	)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.ForbidRoles(constants.RoleTypeHistorian),
		controller.CreateVerification,
	)

}

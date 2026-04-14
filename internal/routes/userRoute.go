package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func UserRoutes(app *fiber.App, controller *controllers.UserController, userRepo repositories.UserRepository) {
	route := app.Group("/users")

	route.Get(
		"/current",
		middlewares.JwtAccess(userRepo),
		controller.GetUserCurrent,
	)

	route.Get(
		"/current/media",
		middlewares.JwtAccess(userRepo),
		controller.GetUserMedia,
	)

	route.Get(
		"/current/application",
		middlewares.JwtAccess(userRepo),
		controller.GetUserApplication,
	)

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.SearchUser,
	)

	route.Put(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.UpdateProfile,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.DeleteUser,
	)

	route.Get(
		"/:id/media",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.GetMediaByUserID,
	)

	route.Get(
		"/:id/application",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.GetVerificationByUserID,
	)

	route.Patch(
		"/:id/restore",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.RestoreUser,
	)

	route.Patch(
		"/:id/role",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN),
		controller.ChangeRoleUser,
	)

	route.Patch(
		"/:id/password",
		middlewares.JwtAccess(userRepo),
		controller.ChangePassword,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.SearchUser,
	)

}

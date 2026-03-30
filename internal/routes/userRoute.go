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
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.Search,
	)
	
	route.Get(
		"/current",
		middlewares.JwtAccess(userRepo),
		controller.GetUserCurrent,
	)
	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.ADMIN, constants.MOD),
		controller.Search,
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
}

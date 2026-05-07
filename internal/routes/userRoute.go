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

	route.Put(
		"/current",
		middlewares.JwtAccess(userRepo),
		controller.UpdateProfile,
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
		"/current/project",
		middlewares.JwtAccess(userRepo),
		controller.GetUserProject,
	)

	route.Patch(
		"/current/password",
		middlewares.JwtAccess(userRepo),
		controller.ChangePassword,
	)

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetUserById,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.DeleteUser,
	)
	
	route.Put(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.AdminUpdateProfile,
	)

	route.Get(
		"/:id/media",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetMediaByUserID,
	)

	route.Get(
		"/:id/application",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetVerificationByUserID,
	)

	route.Get(
		"/:id/project",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetProjectByUserID,
	)

	route.Patch(
		"/:id/restore",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.RestoreUser,
	)

	route.Patch(
		"/:id/password",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.AdminResetPassword,
	)

	route.Patch(
		"/:id/role",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.ChangeRoleUser,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.SearchUser,
	)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.CreateUser,
	)

}

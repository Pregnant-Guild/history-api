package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func MediaRoutes(app *fiber.App, controller *controllers.MediaController, userRepo repositories.UserRepository) {
	route := app.Group("/media")

	route.Post(
		"/upload",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.UploadServerSide,
	)

	route.Get(
		"/presigned",
		middlewares.JwtAccess(userRepo),
		controller.GeneratePresignedURL,
	)

	route.Post(
		"/presigned/complete",
		middlewares.JwtAccess(userRepo),
		controller.PreSignedCompleted,
	)

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.GetMediaByID,
	)

	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		controller.DeleteMedia,
	)

	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.SearchMedia,
	)
	route.Delete(
		"/",
		middlewares.JwtAccess(userRepo),
		controller.BulkDeleteMedia,
	)

}

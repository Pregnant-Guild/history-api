package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"
	"history-api/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func ProjectRoutes(
	app *fiber.App,
	controller *controllers.ProjectController,
	commitController *controllers.CommitController,
	userRepo repositories.UserRepository,
) {
	route := app.Group("/projects")
	
	route.Get(
		"/commits/:commitId",
		middlewares.JwtAccess(userRepo),
		commitController.GetCommitByID,
	)

	route.Post(
		"/:id/commits",
		middlewares.JwtAccess(userRepo),
		commitController.CreateCommit,
	)

	route.Post(
		"/:id/commits/restore",
		middlewares.JwtAccess(userRepo),
		commitController.RestoreCommit,
	)

	route.Get(
		"/:id/commits",
		middlewares.JwtAccess(userRepo),
		commitController.GetProjectCommits,
	)


	route.Post(
		"/:id/members",
		middlewares.JwtAccess(userRepo),
		controller.AddMember,
	)

	route.Put(
		"/:id/members/:userId",
		middlewares.JwtAccess(userRepo),
		controller.UpdateMemberRole,
	)

	route.Delete(
		"/:id/members/:userId",
		middlewares.JwtAccess(userRepo),
		controller.RemoveMember,
	)

	route.Put(
		"/:id/change-owner",
		middlewares.JwtAccess(userRepo),
		controller.ChangeOwner,
	)

	route.Post(
		"/:id/lock",
		middlewares.JwtAccess(userRepo),
		controller.LockProject,
	)

	route.Post(
		"/:id/unlock",
		middlewares.JwtAccess(userRepo),
		controller.UnlockProject,
	)

	route.Post(
		"/:id/heartbeat",
		middlewares.JwtAccess(userRepo),
		controller.HeartbeatProject,
	)

	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
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
		middlewares.JwtAccess(userRepo),
		middlewares.RequireAnyRole(constants.RoleTypeAdmin, constants.RoleTypeMod),
		controller.SearchProject,
	)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		controller.CreateProject,
	)
}

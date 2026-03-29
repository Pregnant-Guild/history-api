package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func AuthRoutes(app *fiber.App, controller *controllers.AuthController, userRepo repositories.UserRepository) {
	route := app.Group("/auth")
	route.Post("/signin", controller.Signin)
	route.Post("/signup", controller.Signup)
	route.Post("/refresh", middlewares.JwtRefresh(userRepo), controller.RefreshToken)
	route.Post("/token/create", controller.CreateToken)
	route.Post("/token/verify", controller.VerifyToken)
	route.Post("/forgot-password", controller.ForgotPassword)
}

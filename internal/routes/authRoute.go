package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"

	"github.com/gofiber/fiber/v3"
)

func AuthRoutes(app *fiber.App, controller *controllers.AuthController) {
	route := app.Group("/auth")
	route.Post("/signin", controller.Signin)
	route.Post("/signup", controller.Signup)
	route.Post("/refresh", middlewares.JwtRefresh(), controller.RefreshToken)
}

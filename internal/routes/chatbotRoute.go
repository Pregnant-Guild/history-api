package routes

import (
	"history-api/internal/controllers"
	"history-api/internal/middlewares"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func ChatbotRoutes(app *fiber.App, controller *controllers.ChatbotController, userRepo repositories.UserRepository) {
	route := app.Group("/chatbot")

	route.Use(middlewares.JwtAccess(userRepo))
	route.Post("/chat", controller.Chat)
}

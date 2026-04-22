package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func EntityRoutes(router fiber.Router, entityController *controllers.EntityController) {
	entity := router.Group("/entities")
	entity.Get("/", entityController.SearchEntities)
	entity.Get("/:id", entityController.GetEntityById)
}

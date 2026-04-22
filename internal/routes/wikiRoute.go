package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func SetupWikiRoutes(router fiber.Router, wikiController *controllers.WikiController) {
	wiki := router.Group("/wikis")
	wiki.Get("/", wikiController.SearchWikis)
	wiki.Get("/:id", wikiController.GetWikiById)
}

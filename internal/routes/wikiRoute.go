package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func WikiRoutes(router fiber.Router, wikiController *controllers.WikiController) {
	wiki := router.Group("/wikis")
	wiki.Get("/", wikiController.SearchWikis)
	wiki.Get("/slug/exists", wikiController.IsExistWikiSlug)
	wiki.Get("/slug/:slug", wikiController.GetWikiBySlug)
	wiki.Get("/:id", wikiController.GetWikiById)
}


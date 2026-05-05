package controllers

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"history-api/pkg/validator"
	"time"

	"github.com/gofiber/fiber/v3"
)

type WikiController struct {
	service services.WikiService
}

func NewWikiController(svc services.WikiService) *WikiController {
	return &WikiController{service: svc}
}

// GetWikiById handles fetching a single wiki by ID.
// @Summary      Get wiki by ID
// @Description  Get detailed information about a specific wiki
// @Tags         Wikis
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Wiki ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /wikis/{id} [get]
func (h *WikiController) GetWikiById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")
	res, err := h.service.GetWikiByID(ctx, id)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// GetWikiBySlug handles fetching a single wiki by slug.
// @Summary      Get wiki by slug
// @Description  Get detailed information about a specific wiki by its unique slug
// @Tags         Wikis
// @Accept       json
// @Produce      json
// @Param        slug   path      string  true  "Wiki Slug"
// @Success      200  {object}  response.CommonResponse
// @Failure      404  {object}  response.CommonResponse
// @Router       /wikis/slug/{slug} [get]
func (h *WikiController) GetWikiBySlug(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slug := c.Params("slug")
	res, err := h.service.GetWikiBySlug(ctx, slug)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// IsExistWikiSlug checks if a wiki slug already exists.
// @Summary      Check wiki slug existence
// @Description  Check if a given slug already exists for wikis
// @Tags         Wikis
// @Accept       json
// @Produce      json
// @Param        slug       query     string  true   "Slug to check"
// @Success      200  {object}  response.CommonResponse
// @Failure      400  {object}  response.CommonResponse
// @Router       /wikis/slug/exists [get]
func (h *WikiController) IsExistWikiSlug(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slug := c.Query("slug")
	exists, err := h.service.IsExistWikiSlug(ctx, slug)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   map[string]bool{"exists": exists},
	})
}

// SearchWikis handles searching for wikis.
// @Summary      Search wikis
// @Description  Search wikis with cursor pagination
// @Tags         Wikis
// @Accept       json
// @Produce      json
// @Param query query request.SearchWikiDto false "Search Query"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /wikis [get]
func (h *WikiController) SearchWikis(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchWikiDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchWikis(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}


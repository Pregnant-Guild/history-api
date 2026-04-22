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
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
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
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

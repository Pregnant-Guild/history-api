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

type EntityController struct {
	service services.EntityService
}

func NewEntityController(svc services.EntityService) *EntityController {
	return &EntityController{service: svc}
}

// GetEntityById handles fetching a single entity by ID.
// @Summary      Get entity by ID
// @Description  Get detailed information about a specific entity
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Entity ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /entities/{id} [get]
func (h *EntityController) GetEntityById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")
	res, err := h.service.GetEntityByID(ctx, id)
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

// SearchEntities handles searching for entities.
// @Summary      Search entities
// @Description  Search entities with cursor pagination
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Param query query request.SearchEntityDto false "Search Query"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /entities [get]
func (h *EntityController) SearchEntities(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchEntityDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchEntities(ctx, dto)
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

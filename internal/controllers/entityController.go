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

// GetEntityBySlug handles fetching a single entity by slug.
// @Summary      Get entity by slug
// @Description  Get detailed information about a specific entity by its unique slug
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Param        slug   path      string  true  "Entity Slug"
// @Success      200  {object}  response.CommonResponse
// @Failure      404  {object}  response.CommonResponse
// @Router       /entities/slug/{slug} [get]
func (h *EntityController) GetEntityBySlug(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slug := c.Params("slug")
	res, err := h.service.GetEntityBySlug(ctx, slug)
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

// IsExistEntitySlug checks if an entity slug already exists.
// @Summary      Check entity slug existence
// @Description  Check if a given slug already exists for entities
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Param        slug       query     string  true   "Slug to check"
// @Success      200  {object}  response.CommonResponse
// @Failure      400  {object}  response.CommonResponse
// @Router       /entities/slug/exists [get]
func (h *EntityController) IsExistEntitySlug(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slug := c.Query("slug")
	exists, err := h.service.IsExistEntitySlug(ctx, slug)
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
	dto := &request.SearchEntityDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchEntities(c.Context(), dto)
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

// GetEntitiesByGeometryIDs handles fetching entities by a list of geometry IDs.
// @Summary      Get entities by geometry IDs
// @Description  Get entities grouped by geometry IDs
// @Tags         Relations
// @Accept       json
// @Produce      json
// @Param query query request.GetEntitiesByGeometryIDsDto true "Query Parameters"
// @Success      200  {object}  response.CommonResponse
// @Failure      400  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /relations/entities-by-geometries [get]
func (h *EntityController) GetEntitiesByGeometryIDs(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.GetEntitiesByGeometryIDsDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.GetEntitiesByGeometryIDs(ctx, dto)
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

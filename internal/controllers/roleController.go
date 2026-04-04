package controllers

import (
	"context"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"time"

	"github.com/gofiber/fiber/v3"
)

type RoleController struct {
	service services.RoleService
}

func NewRoleController(svc services.RoleService) *RoleController {
	return &RoleController{service: svc}
}

// GetRoleById handles fetching a single role by ID.
// @Summary      Get role by ID
// @Description  Get detailed information about a specific role
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /roles/{id} [get]
func (h *RoleController) GetRoleById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	RoleId := c.Params("id")
	res, err := h.service.GetRoleByID(ctx, RoleId)
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


// GetAllRole handles fetching all roles.
// @Summary      Get all roles
// @Description  Get a list of all roles in the system
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /roles [get]
func (h *RoleController) GetAllRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := h.service.GetAllRole(ctx)
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

package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"history-api/pkg/validator"
)

type ProjectController struct {
	service services.ProjectService
}

func NewProjectController(service services.ProjectService) *ProjectController {
	return &ProjectController{
		service: service,
	}
}

// GetProjectByID godoc
// @Summary Get project by ID
// @Description Retrieve project details by specific ID
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id} [get]
func (h *ProjectController) GetProjectByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	res, err := h.service.GetProjectByID(ctx, projectID)
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

// SearchProject godoc
// @Summary Search projects
// @Description Search and filter projects with pagination
// @Tags Projects
// @Accept json
// @Produce json
// @Param query query request.SearchProjectDto false "Search Query"
// @Success 200 {object} response.PaginatedResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects [get]
func (h *ProjectController) SearchProject(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchProjectDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchProject(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// CreateProject godoc
// @Summary Create a new project
// @Description Create a project for the current authenticated user
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateProjectDto true "Project Data"
// @Success 201 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects [post]
func (h *ProjectController) CreateProject(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.CreateProjectDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	uid := c.Locals("uid").(string)
	res, err := h.service.CreateProject(ctx, uid, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// UpdateProject godoc
// @Summary Update a project
// @Description Update project properties (Title, Description, Status)
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body request.UpdateProjectDto true "Project Data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id} [put]
func (h *ProjectController) UpdateProject(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	dto := &request.UpdateProjectDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.UpdateProject(ctx, projectID, dto)
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

// DeleteProject godoc
// @Summary Delete a project
// @Description Delete project by ID
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id} [delete]
func (h *ProjectController) DeleteProject(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	err := h.service.DeleteProject(ctx, projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Project deleted successfully",
	})
}

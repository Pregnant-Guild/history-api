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
// @Security BearerAuth
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

// SearchProject godoc
// @Summary Search projects
// @Description Search and filter projects with pagination
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
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
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
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
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
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
	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}
	res, err := h.service.UpdateProject(ctx, claims, projectID, dto)
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

	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}
	err := h.service.DeleteProject(ctx, claims, projectID)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Project deleted successfully",
	})
}

// AddMember godoc
// @Summary Add a member to project
// @Description Invite a user to the project with a specific role (EDITOR or VIEWER). Only project owner can do this.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body request.AddProjectMemberDto true "Member Data"
// @Success 201 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 409 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/members [post]
func (h *ProjectController) AddMember(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	dto := &request.AddProjectMemberDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}

	res, err := h.service.AddMember(ctx, claims, projectID, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// UpdateMemberRole godoc
// @Summary Update member role
// @Description Change a member's role in the project. Only project owner can do this.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param userId path string true "Member User ID"
// @Param request body request.UpdateProjectMemberDto true "Role Data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/members/{userId} [put]
func (h *ProjectController) UpdateMemberRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	memberUserID := c.Params("userId")
	dto := &request.UpdateProjectMemberDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}

	res, err := h.service.UpdateMemberRole(ctx, claims, projectID, memberUserID, dto)
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

// RemoveMember godoc
// @Summary Remove a member from project
// @Description Remove a user from the project. Only project owner can do this.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param userId path string true "Member User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/members/{userId} [delete]
func (h *ProjectController) RemoveMember(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	memberUserID := c.Params("userId")

	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}

	err := h.service.RemoveMember(ctx, claims, projectID, memberUserID)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Member removed successfully",
	})
}

// ChangeOwner godoc
// @Summary Transfer project ownership
// @Description Transfer project ownership to an existing member. Only the current owner can do this.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body request.ChangeOwnerDto true "New Owner Data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/change-owner [put]
func (h *ProjectController) ChangeOwner(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	dto := &request.ChangeOwnerDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Unauthorized",
		})
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid user claims",
		})
	}
	res, err := h.service.ChangeOwner(ctx, claims, projectID, dto.NewOwnerID)
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

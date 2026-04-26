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

type CommitController struct {
	service services.CommitService
}

func NewCommitController(service services.CommitService) *CommitController {
	return &CommitController{
		service: service,
	}
}

// CreateCommit godoc
// @Summary Create a new commit
// @Description Create a new commit and update project's latest commit ID. Only owner/editor allowed.
// @Tags Commits
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body request.CreateCommitDto true "Commit Data"
// @Success 201 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/commits [post]
func (h *CommitController) CreateCommit(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	dto := &request.CreateCommitDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	uid := c.Locals("uid").(string)
	res, err := h.service.CreateCommit(ctx, uid, projectID, dto)
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

// RestoreCommit godoc
// @Summary Restore project to a commit
// @Description Update project's latest commit ID to an older commit. Only owner/editor allowed.
// @Tags Commits
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body request.RestoreCommitDto true "Restore Data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/commits/restore [post]
func (h *CommitController) RestoreCommit(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	dto := &request.RestoreCommitDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	uid := c.Locals("uid").(string)
	err := h.service.RestoreCommit(ctx, uid, projectID, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Project restored successfully",
	})
}

// GetProjectCommits godoc
// @Summary Get project commits
// @Description Retrieve all commits for a specific project
// @Tags Commits
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /projects/{id}/commits [get]
func (h *CommitController) GetProjectCommits(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectID := c.Params("id")
	res, err := h.service.GetProjectCommits(ctx, projectID)
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

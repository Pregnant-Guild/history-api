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

type SubmissionController interface {
	CreateSubmission(c fiber.Ctx) error
	UpdateSubmissionStatus(c fiber.Ctx) error
	GetSubmissionByID(c fiber.Ctx) error
	SearchSubmissions(c fiber.Ctx) error
	DeleteSubmission(c fiber.Ctx) error
}

type submissionController struct {
	submissionService services.SubmissionService
}

func NewSubmissionController(submissionService services.SubmissionService) SubmissionController {
	return &submissionController{
		submissionService: submissionService,
	}
}

// @Summary Create submission
// @Description Submit a new submission for a project commit
// @Tags Submission
// @Accept json
// @Produce json
// @Param body body request.CreateSubmissionDto true "Submission data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /submissions [post]
func (s *submissionController) CreateSubmission(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.CreateSubmissionDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := s.submissionService.CreateSubmission(ctx, c.Locals("uid").(string), dto)
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

// @Summary Update submission status
// @Description Approve or reject a submission
// @Tags Submission
// @Accept json
// @Produce json
// @Param id path string true "Submission ID"
// @Param body body request.UpdateSubmissionStatusDto true "Status update data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /submissions/{id}/status [patch]
func (s *submissionController) UpdateSubmissionStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	id := c.Params("id")
	uid := c.Locals("uid").(string)
	dto := &request.UpdateSubmissionStatusDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := s.submissionService.UpdateSubmissionStatus(ctx, uid, id, dto)
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

// @Summary Get submission by ID
// @Description Get detailed information of a submission
// @Tags Submission
// @Produce json
// @Param id path string true "Submission ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /submissions/{id} [get]
func (s *submissionController) GetSubmissionByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")
	res, err := s.submissionService.GetSubmissionByID(ctx, id)
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

// @Summary Search submissions
// @Description Get a list of submissions with filters
// @Tags Submission
// @Accept json
// @Produce json
// @Param query query request.SearchSubmissionDto false "Search Query"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /submissions [get]
func (s *submissionController) SearchSubmissions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.SearchSubmissionDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := s.submissionService.SearchSubmissions(ctx, dto)
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

// @Summary Delete submission
// @Description Delete a submission by ID
// @Tags Submission
// @Produce json
// @Param id path string true "Submission ID"
// @Success 200 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /submissions/{id} [delete]
func (s *submissionController) DeleteSubmission(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")

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

	err := s.submissionService.DeleteSubmission(ctx, c.Locals("uid").(string), id, claims)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Submission deleted successfully",
	})
}

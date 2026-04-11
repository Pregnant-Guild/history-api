package controllers

import (
	"context"
	"history-api/internal/dtos/response"
	"history-api/internal/dtos/request"
	"history-api/internal/services"
	"history-api/pkg/validator"
	"time"

	"github.com/gofiber/fiber/v3"
)

type VerificationController struct {
	service services.VerificationService
}

func NewVerificationController(svc services.VerificationService) *VerificationController {
	return &VerificationController{service: svc}
}

// @Summary Get application by ID
// @Description Get historian application detail
// @Tags Historian Application
// @Produce json
// @Param id path string true "Verification ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /historian/application/{id} [get]
func (m *VerificationController) GetVerificationByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	verificationId := c.Params("id")
	res, err := m.service.GetVerificationByID(ctx, verificationId)
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

// @Summary Search historian applications
// @Description Get list of historian applications with filters
// @Tags Historian Application
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query request.SearchUserVerificationDto false "Search Query"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /historian/application [get]
func (m *VerificationController) SearchVerification(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchUserVerificationDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := m.service.SearchVerification(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// @Summary Delete application
// @Description Delete historian application
// @Tags Historian Application
// @Produce json
// @Param id path string true "Verification ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /historian/application/{id} [delete]
func (m *VerificationController) DeleteVerification(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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

	verificationId := c.Params("id")
	err := m.service.DeleteVerification(ctx, claims, verificationId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Verification deleted successfully",
	})
}

// @Summary Create historian application
// @Description Submit application to become historian
// @Tags Historian Application
// @Accept json
// @Produce json
// @Param body body request.CreateUserVerificationDto true "Application data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /historian/application [post]
func (m *VerificationController) CreateVerification(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.CreateUserVerificationDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := m.service.CreateVerification(ctx, c.Locals("uid").(string), dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// @Summary Update application status
// @Description Approve or reject historian application
// @Tags Historian Application
// @Accept json
// @Produce json
// @Param id path string true "Verification ID"
// @Param body body request.UpdateVerificationStatusDto true "Status update"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Security BearerAuth
// @Router /historian/application/{id}/status [put]
func (m *VerificationController) UpdateVerificationStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateVerificationStatusDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	verificationId := c.Params("id")
	res, err := m.service.UpdateStatusVerification(ctx, c.Locals("uid").(string), verificationId, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

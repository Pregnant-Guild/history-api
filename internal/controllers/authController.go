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

type AuthController struct {
	service services.AuthService
}

func NewAuthController(svc services.AuthService) *AuthController {
	return &AuthController{service: svc}
}

// Signin godoc
// @Summary Sign in a user
// @Description Authenticate user credentials and return access/refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.SignInDto true "Sign In credentials"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse "Invalid credentials"
// @Failure 500 {object} response.CommonResponse
// @Router /auth/signin [post]
func (h *AuthController) Signin(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.SignInDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.Signin(ctx, dto)
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

// Signup godoc
// @Summary Register a new user
// @Description Create a new user account in the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.SignUpDto true "Sign Up details"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/signup [post]
func (h *AuthController) Signup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.SignUpDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.Signup(ctx, dto)
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

// RefreshToken godoc
// @Summary Refresh session tokens
// @Description Generate a new access token using a valid refresh token from context
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse "Unauthorized or expired refresh token"
// @Failure 500 {object} response.CommonResponse
// @Router /auth/refresh [post]
func (h *AuthController) RefreshToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.RefreshToken(ctx, c.Locals("uid").(string))
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

// VerifyToken godoc
// @Summary Verify a security token
// @Description Validate an OTP or email verification token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.VerifyTokenDto true "Token verification data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/token/verify [post]
func (h *AuthController) VerifyToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.VerifyTokenDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.VerifyToken(ctx, dto)
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

// CreateToken godoc
// @Summary Generate a new verification token
// @Description Request a new token for specific actions like email confirmation
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.CreateTokenDto true "Token creation request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/token/create [post]
func (h *AuthController) CreateToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.CreateTokenDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	err := h.service.CreateToken(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Data:    nil,
		Message: "Token created successfully",
	})
}

// ForgotPassword godoc
// @Summary Handle forgotten password
// @Description Initiate password recovery process for a user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.ForgotPasswordDto true "Forgot Password request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/forgot-password [post]
func (h *AuthController) ForgotPassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.ForgotPasswordDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	err := h.service.ForgotPassword(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Data:    nil,
		Message: "Password reset successfully",
	})
}
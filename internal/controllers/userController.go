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

type UserController struct {
	service      services.UserService
	mediaService services.MediaService
	verificationService services.VerificationService
}

func NewUserController(
	svc services.UserService, 
	mediaSvc services.MediaService, 
	verificationSvc services.VerificationService,
) *UserController {
	return &UserController{
		service:      svc,
		mediaService: mediaSvc,
		verificationService: verificationSvc,
	}
}

// GetUserCurrent godoc
// @Summary Get current user profile
// @Description Retrieve the profile information of the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/current [get]
func (h *UserController) GetUserCurrent(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetUserCurrent(ctx, c.Locals("uid").(string))
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

// GetUserMedia godoc
// @Summary Get current user's media
// @Description Retrieve media list of the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/current/media [get]
func (h *UserController) GetUserMedia(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.mediaService.GetMediaByUserID(ctx, c.Locals("uid").(string))
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

// GetMediaByUserID godoc
// @Summary Get user's media by user ID
// @Description Retrieve media list by specific user ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id}/media [get]
func (h *UserController) GetMediaByUserID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Params("id")
	res, err := h.mediaService.GetMediaByUserID(ctx, userId)
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

// GetApplicationUserID godoc
// @Summary Get user's media by user ID
// @Description Retrieve media list by specific user ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id}/application [get]
func (h *UserController) GetVerificationByUserID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Params("id")
	res, err := h.verificationService.GetVerificationByUserID(ctx, userId)
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

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update the profile details of the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body request.UpdateProfileDto true "Update Profile request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id} [put]
func (h *UserController) UpdateProfile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateProfileDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.UpdateProfile(ctx, c.Locals("uid").(string), dto)
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

// ChangePassword godoc
// @Summary Change user password
// @Description Update the password for the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body request.ChangePasswordDto true "Change Password request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id}/password [patch]
func (h *UserController) ChangePassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.ChangePasswordDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	err := h.service.ChangePassword(ctx, c.Locals("uid").(string), dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Password changed successfully",
	})
}

// RestoreUser godoc
// @Summary Restore a deleted user
// @Description Restore a soft-deleted user account (Admin/Mod only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id}/restore [patch]
func (h *UserController) RestoreUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Params("id")
	res, err := h.service.RestoreUser(ctx, userId)
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

// DeleteUser godoc
// @Summary Delete a user
// @Description Soft delete a user account (Admin/Mod only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id} [delete]
func (h *UserController) DeleteUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Params("id")
	err := h.service.DeleteUser(ctx, userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "User deleted successfully",
	})
}

// ChangeRoleUser godoc
// @Summary Change user role
// @Description Update the role of a user (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body request.ChangeRoleDto true "Change Role request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id}/role [patch]
func (h *UserController) ChangeRoleUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ChangeRoleDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
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

	user, err := h.service.ChangeRoleUser(ctx, claims, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   user,
	})
}

// GetUserById godoc
// @Summary Get user by ID
// @Description Retrieve details of a specific user (Admin/Mod only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users/{id} [get]
func (h *UserController) GetUserById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Params("id")
	res, err := h.service.GetUserByID(ctx, userId)
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

// Search godoc
// @Summary Search users
// @Description Search and filter users with pagination (Admin/Mod only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query request.SearchUserDto false "Search Query"
// @Success 200 {object} response.PaginatedResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /users [get]
func (h *UserController) SearchUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchUserDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := h.service.SearchUser(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

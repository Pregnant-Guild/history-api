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

type MediaController struct {
	service services.MediaService
}

func NewMediaController(svc services.MediaService) *MediaController {
	return &MediaController{service: svc}
}

// GetMediaByID godoc
// @Summary Get media by ID
// @Description Retrieve a media file by its ID
// @Tags Media
// @Accept json
// @Produce json
// @Param id path string true "Media ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media/{id} [get]
func (m *MediaController) GetMediaByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mediaId := c.Params("id")
	res, err := m.service.GetMediaByID(ctx, mediaId)
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

// SearchMedia godoc
// @Summary Search media
// @Description Search media with filters, pagination
// @Tags Media
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param keyword query string false "Search keyword"
// @Success 200 {object} response.PaginatedResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media [get]
func (m *MediaController) SearchMedia(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchMediaDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := m.service.SearchMedia(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// DeleteMedia godoc
// @Summary Delete media
// @Description Delete a media file by ID
// @Tags Media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Media ID"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media/{id} [delete]
func (m *MediaController) DeleteMedia(c fiber.Ctx) error {
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

	mediaId := c.Params("id")
	err := m.service.DeleteMedia(ctx, claims, mediaId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Media deleted successfully",
	})
}

// BulkDeleteMedia godoc
// @Summary Delete media
// @Description Delete multiple media files by IDs
// @Tags Media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body request.MediaBulkDeleteDto true "Media IDs to delete"
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media [delete]
func (m *MediaController) BulkDeleteMedia(c fiber.Ctx) error {
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

	dto := &request.MediaBulkDeleteDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	err := m.service.BulkDeleteMedia(ctx, claims, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Media deleted successfully",
	})
}

// UploadServerSide godoc
// @Summary Upload media (server-side)
// @Description Upload media file through server
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Upload file"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media/upload [post]
func (m *MediaController) UploadServerSide(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "File is required",
		})
	}

	url, err := m.service.UploadServerSide(ctx, c.Locals("uid").(string), fileHeader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Data:    url,
		Message: "Media uploaded successfully",
	})
}

// GeneratePresignedURL godoc
// @Summary Generate presigned URL
// @Description Generate a presigned URL for direct upload to storage
// @Tags Media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fileName query string true "File name"
// @Param content_type query string true "Content type"
// @Param size query int true "File size"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media/presigned [get]
func (m *MediaController) GeneratePresignedURL(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.PreSignedDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := m.service.GeneratePresignedURL(ctx, c.Locals("uid").(string), dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// PreSignedCompleted godoc
// @Summary Confirm presigned upload
// @Description Confirm that upload via presigned URL is completed
// @Tags Media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body PreSignedCompleteDto true "Request body"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /media/presigned/complete [post]
func (m *MediaController) PreSignedCompleted(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.PreSignedCompleteDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	res, err := m.service.PreSignedCompleted(ctx, c.Locals("uid").(string), dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

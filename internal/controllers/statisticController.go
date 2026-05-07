package controllers

import (
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"history-api/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

type StatisticController struct {
	statService services.StatisticService
}

func NewStatisticController(statService services.StatisticService) *StatisticController {
	return &StatisticController{
		statService: statService,
	}
}

// SearchStatistics godoc
// @Summary Search system statistics
// @Description Fetch daily system statistics with optional date range filtering
// @Tags Statistics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date in YYYY-MM-DD format"
// @Param end_date query string false "End date in YYYY-MM-DD format"
// @Success 200 {object} response.CommonResponse{data=[]response.StatisticResponse}
// @Failure 400 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /statistics [get]
// @Security BearerAuth
func (c *StatisticController) SearchStatistics(ctx fiber.Ctx) error {
	dto := new(request.SearchStatisticDto)
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := c.statService.Search(ctx.Context(), dto)
	if err != nil {
		return ctx.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// GetStatisticByDate godoc
// @Summary Get system statistics by date
// @Description Fetch system statistics for a specific date
// @Tags Statistics
// @Accept json
// @Produce json
// @Param date path string true "Date in YYYY-MM-DD format"
// @Success 200 {object} response.CommonResponse{data=response.StatisticResponse}
// @Failure 400 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse
// @Failure 403 {object} response.CommonResponse
// @Failure 404 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /statistics/{date} [get]
// @Security BearerAuth
func (c *StatisticController) GetStatisticByDate(ctx fiber.Ctx) error {
	dateStr := ctx.Params("date")
	if dateStr == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "Date parameter is required",
		})
	}

	res, err := c.statService.GetByDate(ctx.Context(), dateStr)
	if err != nil {
		return ctx.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

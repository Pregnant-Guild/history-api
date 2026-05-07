package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/repositories"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
)

type StatisticService interface {
	Search(ctx context.Context, dto *request.SearchStatisticDto) ([]*response.StatisticResponse, *fiber.Error)
	GetByDate(ctx context.Context, dateStr string) (*response.StatisticResponse, *fiber.Error)
}

type statisticService struct {
	repo repositories.StatisticRepository
}

func NewStatisticService(repo repositories.StatisticRepository) StatisticService {
	return &statisticService{repo: repo}
}

func (s *statisticService) Search(ctx context.Context, dto *request.SearchStatisticDto) ([]*response.StatisticResponse, *fiber.Error) {
	params := sqlc.SearchSystemStatisticsParams{}

	if dto.StartDate != "" {
		parsedDate, err := time.Parse("2006-01-02", dto.StartDate)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid start_date format, expected YYYY-MM-DD")
		}
		params.StartDate = pgtype.Date{Time: parsedDate, Valid: true}
	}

	if dto.EndDate != "" {
		parsedDate, err := time.Parse("2006-01-02", dto.EndDate)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid end_date format, expected YYYY-MM-DD")
		}
		params.EndDate = pgtype.Date{Time: parsedDate, Valid: true}
	}

	stats, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search statistics")
	}

	responses := make([]*response.StatisticResponse, 0, len(stats))
	for _, stat := range stats {
		responses = append(responses, stat.ToResponse())
	}

	return responses, nil
}

func (s *statisticService) GetByDate(ctx context.Context, dateStr string) (*response.StatisticResponse, *fiber.Error) {
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid date format, expected YYYY-MM-DD")
	}

	stat, err := s.repo.GetByDate(ctx, parsedDate)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get statistics")
	}
	if stat == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Statistics not found for the given date")
	}

	return stat.ToResponse(), nil
}

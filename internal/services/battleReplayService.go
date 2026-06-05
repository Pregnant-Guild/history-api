package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
)

type BattleReplayService interface {
	GetByID(ctx context.Context, id string) (*response.BattleReplayResponse, *fiber.Error)
	GetByGeometryID(ctx context.Context, geometryID string) ([]*response.BattleReplayResponse, *fiber.Error)
	GetByGeometryIDs(ctx context.Context, req *request.GetBattleReplaysByGeometryIDsDto) (map[string][]*response.BattleReplayResponse, *fiber.Error)
}

type battleReplayService struct {
	battleReplayRepo repositories.BattleReplayRepository
}

func NewBattleReplayService(battleReplayRepo repositories.BattleReplayRepository) BattleReplayService {
	return &battleReplayService{
		battleReplayRepo: battleReplayRepo,
	}
}

func (s *battleReplayService) GetByID(ctx context.Context, id string) (*response.BattleReplayResponse, *fiber.Error) {
	replayUUID, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid battle replay ID format")
	}

	replay, err := s.battleReplayRepo.GetByID(ctx, replayUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Battle replay not found")
	}

	return replay.ToResponse(), nil
}

func (s *battleReplayService) GetByGeometryID(ctx context.Context, geometryID string) ([]*response.BattleReplayResponse, *fiber.Error) {
	geomUUID, err := convert.StringToUUID(geometryID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid geometry ID format")
	}

	replays, err := s.battleReplayRepo.GetByGeometryID(ctx, geomUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get battle replays")
	}

	return models.BattleReplaysEntityToResponse(replays), nil
}

func (s *battleReplayService) GetByGeometryIDs(ctx context.Context, req *request.GetBattleReplaysByGeometryIDsDto) (map[string][]*response.BattleReplayResponse, *fiber.Error) {
	replays, err := s.battleReplayRepo.GetByGeometryIDs(ctx, req.GeometryIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get battle replays")
	}

	counts := make(map[string]int, len(req.GeometryIDs))
	for _, replay := range replays {
		if replay != nil {
			counts[replay.GeometryID]++
		}
	}

	result := make(map[string][]*response.BattleReplayResponse, len(req.GeometryIDs))
	for _, idStr := range req.GeometryIDs {
		result[idStr] = make([]*response.BattleReplayResponse, 0, counts[idStr])
	}

	for _, replay := range replays {
		if replay != nil {
			geomID := replay.GeometryID
			if _, exists := result[geomID]; exists {
				result[geomID] = append(result[geomID], replay.ToResponse())
			}
		}
	}

	return result, nil
}

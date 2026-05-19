package services

import (
	"context"
	"encoding/json"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommitService interface {
	CreateCommit(ctx context.Context, userID string, projectID string, dto *request.CreateCommitDto) (*response.CommitResponse, *fiber.Error)
	RestoreCommit(ctx context.Context, userID string, projectID string, dto *request.RestoreCommitDto) *fiber.Error
	GetProjectCommits(ctx context.Context, projectID string) ([]*response.CommitResponse, *fiber.Error)
	GetCommitByID(ctx context.Context, commitID string) (*response.CommitResponse, *fiber.Error)
}

type commitService struct {
	db          *pgxpool.Pool
	commitRepo  repositories.CommitRepository
	projectRepo repositories.ProjectRepository
	c           cache.Cache
}

func NewCommitService(
	db *pgxpool.Pool,
	commitRepo repositories.CommitRepository,
	projectRepo repositories.ProjectRepository,
	c cache.Cache,
) CommitService {
	return &commitService{
		db:          db,
		commitRepo:  commitRepo,
		projectRepo: projectRepo,
		c:           c,
	}
}

func (s *commitService) checkWritePermission(ctx context.Context, userID string, projectUUID pgtype.UUID) *fiber.Error {
	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Project not found")
	}

	if project.UserID == userID {
		return nil
	}

	userUUID, _ := convert.StringToUUID(userID)
	roleVal, err := s.projectRepo.CheckPermission(ctx, sqlc.CheckProjectPermissionParams{
		ProjectID: projectUUID,
		UserID:    userUUID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "You do not have permission to write to this project")
	}

	role := constants.ParseProjectMemberRole(roleVal)
	if !role.CanWrite() {
		return fiber.NewError(fiber.StatusForbidden, "You do not have permission to write to this project")
	}

	for _, s := range project.Submissions {
		if s.Status == constants.StatusTypePending {
			return fiber.NewError(fiber.StatusConflict, "Cannot create commit while there is a pending submission")
		}
	}

	return nil
}

func (s *commitService) CreateCommit(ctx context.Context, userID string, projectID string, dto *request.CreateCommitDto) (*response.CommitResponse, *fiber.Error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	cRepoTx := s.commitRepo.WithTx(tx)
	pRepoTx := s.projectRepo.WithTx(tx)

	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	if fErr := s.checkWritePermission(ctx, userID, projectUUID); fErr != nil {
		return nil, fErr
	}

	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	snapshotJSON, err := json.Marshal(dto.SnapshotJson)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid snapshot JSON")
	}

	commit, err := cRepoTx.Create(ctx, sqlc.CreateCommitParams{
		ProjectID:    projectUUID,
		SnapshotJson: snapshotJSON,
		UserID:       userUUID,
		EditSummary:  convert.StringToText(dto.EditSummary),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create commit")
	}

	commitUUID, err := convert.StringToUUID(commit.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to process commit ID")
	}

	err = pRepoTx.UpdateLatestCommit(ctx, sqlc.UpdateLatestCommitParams{
		ID:             projectUUID,
		LatestCommitID: commitUUID,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update project latest commit")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction")
	}

	_ = s.c.Del(ctx, fmt.Sprintf("project:id:%s", projectID), fmt.Sprintf("commit:project:%s", projectID))

	return commit.ToResponse(), nil
}

func (s *commitService) RestoreCommit(ctx context.Context, userID string, projectID string, dto *request.RestoreCommitDto) *fiber.Error {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	if fErr := s.checkWritePermission(ctx, userID, projectUUID); fErr != nil {
		return fErr
	}

	commitUUID, err := convert.StringToUUID(dto.CommitID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
	}

	commit, err := s.commitRepo.GetByID(ctx, commitUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Commit not found")
	}

	if commit.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "Commit does not belong to this project")
	}

	err = s.projectRepo.UpdateLatestCommit(ctx, sqlc.UpdateLatestCommitParams{
		ID:             projectUUID,
		LatestCommitID: commitUUID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to restore commit")
	}

	_ = s.c.Del(ctx, fmt.Sprintf("project:id:%s", projectID), fmt.Sprintf("commit:project:%s", projectID))
	return nil
}

func (s *commitService) GetProjectCommits(ctx context.Context, projectID string) ([]*response.CommitResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	commits, err := s.commitRepo.GetByProjectID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch commits")
	}

	return models.CommitsEntityToResponse(commits), nil
}

func (s *commitService) GetCommitByID(ctx context.Context, commitID string) (*response.CommitResponse, *fiber.Error) {
	commitUUID, err := convert.StringToUUID(commitID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
	}

	commit, err := s.commitRepo.GetByID(ctx, commitUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Commit not found")
	}

	return commit.ToResponse(), nil
}

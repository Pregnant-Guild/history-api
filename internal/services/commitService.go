package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/constants"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommitService interface {
	CreateCommit(ctx context.Context, userID string, projectID string, dto *request.CreateCommitDto) (*response.CommitResponse, error)
	RestoreCommit(ctx context.Context, userID string, projectID string, dto *request.RestoreCommitDto) error
	GetProjectCommits(ctx context.Context, projectID string) ([]*response.CommitResponse, error)
}

type commitService struct {
	db          *pgxpool.Pool
	commitRepo  repositories.CommitRepository
	projectRepo repositories.ProjectRepository
}

func NewCommitService(
	db *pgxpool.Pool,
	commitRepo repositories.CommitRepository,
	projectRepo repositories.ProjectRepository,
) CommitService {
	return &commitService{
		db:          db,
		commitRepo:  commitRepo,
		projectRepo: projectRepo,
	}
}
func (s *commitService) checkWritePermission(ctx context.Context, userID string, projectUUID pgtype.UUID) error {
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

	return nil
}

func (s *commitService) CreateCommit(ctx context.Context, userID string, projectID string, dto *request.CreateCommitDto) (*response.CommitResponse, error) {
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

	if err := s.checkWritePermission(ctx, userID, projectUUID); err != nil {
		return nil, err
	}

	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	commit, err := cRepoTx.Create(ctx, sqlc.CreateCommitParams{
		ProjectID:    projectUUID,
		SnapshotJson: dto.SnapshotJson,
		UserID:       userUUID,
		EditSummary:  convert.StringToText(dto.EditSummary),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create commit")
	}

	commitUUID, err := convert.StringToUUID(commit.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
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

	return commit.ToResponse(), nil
}

func (s *commitService) RestoreCommit(ctx context.Context, userID string, projectID string, dto *request.RestoreCommitDto) error {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	if err := s.checkWritePermission(ctx, userID, projectUUID); err != nil {
		return err
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
	return nil
}

func (s *commitService) GetProjectCommits(ctx context.Context, projectID string) ([]*response.CommitResponse, error) {
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

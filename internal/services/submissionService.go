package services

import (
	"context"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, userID string, dto *request.CreateSubmissionDto) (*response.SubmissionResponse, *fiber.Error)
	UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error)
	GetSubmissionByID(ctx context.Context, id string) (*response.SubmissionResponse, *fiber.Error)
	SearchSubmissions(ctx context.Context, dto *request.SearchSubmissionDto) (*response.PaginatedResponse, *fiber.Error)
	DeleteSubmission(ctx context.Context, userID string, id string, claims *response.JWTClaims) *fiber.Error
}

type submissionService struct {
	submissionRepo repositories.SubmissionRepository
	projectRepo    repositories.ProjectRepository
	commitRepo     repositories.CommitRepository
	userRepo       repositories.UserRepository
	db             *pgxpool.Pool
	c              cache.Cache
}

func NewSubmissionService(
	submissionRepo repositories.SubmissionRepository,
	projectRepo repositories.ProjectRepository,
	commitRepo repositories.CommitRepository,
	userRepo repositories.UserRepository,
	db *pgxpool.Pool,
	c cache.Cache,
) SubmissionService {
	return &submissionService{
		submissionRepo: submissionRepo,
		projectRepo:    projectRepo,
		commitRepo:     commitRepo,
		userRepo:       userRepo,
		db:             db,
		c:              c,
	}
}

func (s *submissionService) CreateSubmission(ctx context.Context, userID string, dto *request.CreateSubmissionDto) (*response.SubmissionResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(dto.ProjectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	commitUUID, err := convert.StringToUUID(dto.CommitID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
	}

	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	commit, err := s.commitRepo.GetByID(ctx, commitUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Commit not found")
	}
	if commit.ProjectID != dto.ProjectID {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Commit does not belong to project")
	}

	arg := sqlc.CreateSubmissionParams{
		ProjectID: projectUUID,
		CommitID:  commitUUID,
		UserID:    userUUID,
		Status:    constants.StatusTypePending.Int16(),
		Content:   convert.StringToText(dto.Content),
	}

	submission, err := s.submissionRepo.Create(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create submission")
	}

	return submission.ToResponse(), nil
}

func (s *submissionService) UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error) {
	submissionUUID, err := convert.StringToUUID(submissionID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	reviewerUUID, err := convert.StringToUUID(reviewerID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid reviewer ID")
	}

	status := constants.ParseStatusTypeText(dto.Status)
	if status == constants.StatusTypeUnknown {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid status")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	if submission.Status != constants.StatusTypePending {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Submission already processed")
	}

	arg := sqlc.UpdateSubmissionParams{
		ID:         submissionUUID,
		Status:     pgtype.Int2{Int16: status.Int16(), Valid: true},
		ReviewedBy: reviewerUUID,
		ReviewNote: convert.StringToText(dto.ReviewNote),
	}

	updatedSubmission, err := s.submissionRepo.Update(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update submission status")
	}

	_ = s.c.Del(ctx, fmt.Sprintf("project:id:%s", submission.ProjectID))

	return updatedSubmission.ToResponse(), nil
}

func (m *submissionService) fillSearchArgs(arg *sqlc.SearchSubmissionsParams, dto *request.SearchSubmissionDto) {
	if dto.Sort != "" {
		arg.Sort = pgtype.Text{String: dto.Sort, Valid: true}
	} else {
		arg.Sort = pgtype.Text{String: "id", Valid: true}
	}

	arg.Order = pgtype.Text{String: "asc", Valid: true}
	if dto.Order == "desc" {
		arg.Order = pgtype.Text{String: "desc", Valid: true}
	}

	if len(dto.Statuses) > 0 {
		for _, id := range dto.Statuses {
			if u := constants.ParseStatusTypeText(id); u != constants.StatusTypeUnknown {
				arg.Statuses = append(arg.Statuses, u.Int16())
			}
		}
	}

	if len(dto.UserIDs) > 0 {
		for _, id := range dto.UserIDs {
			if u, err := convert.StringToUUID(id); err == nil {
				arg.UserIds = append(arg.UserIds, u)
			}
		}
	}

	if dto.ReviewedBy != nil {
		if rvID, err := convert.StringToUUID(*dto.ReviewedBy); err == nil {
			arg.ReviewedBy = rvID
		}
	}

	if dto.CreatedFrom != nil {
		arg.CreatedFrom = pgtype.Timestamptz{Time: *dto.CreatedFrom, Valid: true}
	}

	if dto.CreatedTo != nil {
		arg.CreatedTo = pgtype.Timestamptz{Time: *dto.CreatedTo, Valid: true}
	}

	if dto.Search != "" {
		arg.SearchText = pgtype.Text{String: dto.Search, Valid: true}
	}
}

func (s *submissionService) GetSubmissionByID(ctx context.Context, id string) (*response.SubmissionResponse, *fiber.Error) {
	submissionUUID, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	return submission.ToResponse(), nil
}

func (s *submissionService) SearchSubmissions(ctx context.Context, dto *request.SearchSubmissionDto) (*response.PaginatedResponse, *fiber.Error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit

	arg := sqlc.SearchSubmissionsParams{
		Limit:  int32(dto.Limit),
		Offset: int32(offset),
	}

	s.fillSearchArgs(&arg, dto)

	var rows []*models.SubmissionEntity
	var totalRecords int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = s.submissionRepo.Search(gCtx, arg)
		return err
	})

	g.Go(func() error {
		countArg := sqlc.CountSubmissionsParams{
			UserIds:     arg.UserIds,
			Statuses:    arg.Statuses,
			ReviewedBy:  arg.ReviewedBy,
			CreatedFrom: arg.CreatedFrom,
			CreatedTo:   arg.CreatedTo,
			SearchText:  arg.SearchText,
		}
		var err error
		totalRecords, err = s.submissionRepo.Count(gCtx, countArg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search submissions")
	}

	submissions := models.SubmissionsEntityToResponse(rows)

	return response.BuildPaginatedResponse(submissions, totalRecords, dto.Page, dto.Limit), nil
}

func (s *submissionService) DeleteSubmission(ctx context.Context, userID string, id string, claims *response.JWTClaims) *fiber.Error {
	submissionUUID, err := convert.StringToUUID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	shoudDelete := false
	if slices.Contains(claims.Roles, constants.RoleTypeAdmin) || slices.Contains(claims.Roles, constants.RoleTypeMod) {
		shoudDelete = true
	}

	if submission.UserID == claims.UId && submission.Status == constants.StatusTypePending {
		shoudDelete = true
	}

	if !shoudDelete {
		return fiber.NewError(fiber.StatusForbidden, "You don't have permission to delete this submission")
	}

	err = s.submissionRepo.Delete(ctx, submissionUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete submission")
	}

	return nil
}

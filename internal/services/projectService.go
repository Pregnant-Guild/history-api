package services

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
)

type ProjectService interface {
	GetProjectByID(ctx context.Context, id string) (*response.ProjectResponse, *fiber.Error)
	GetProjectByUserID(ctx context.Context, userID string, dto *request.GetProjectsByUserDto) ([]*response.ProjectResponse, *fiber.Error)
	SearchProject(ctx context.Context, dto *request.SearchProjectDto) (*response.PaginatedResponse, *fiber.Error)
	DeleteProject(ctx context.Context, claims *response.JWTClaims, id string) *fiber.Error
	CreateProject(ctx context.Context, userID string, dto *request.CreateProjectDto) (*response.ProjectResponse, *fiber.Error)
	UpdateProject(ctx context.Context, claims *response.JWTClaims, id string, dto *request.UpdateProjectDto) (*response.ProjectResponse, *fiber.Error)
	AddMember(ctx context.Context, claims *response.JWTClaims, projectID string, dto *request.AddProjectMemberDto) (*response.ProjectResponse, *fiber.Error)
	UpdateMemberRole(ctx context.Context, claims *response.JWTClaims, projectID string, memberUserID string, dto *request.UpdateProjectMemberDto) (*response.ProjectResponse, *fiber.Error)
	RemoveMember(ctx context.Context, claims *response.JWTClaims, projectID string, memberUserID string) *fiber.Error
	ChangeOwner(ctx context.Context, claims *response.JWTClaims, projectID string, newOwnerID string) (*response.ProjectResponse, *fiber.Error)
}

type projectService struct {
	projectRepo repositories.ProjectRepository
}

func NewProjectService(projectRepo repositories.ProjectRepository) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
	}
}

func (s *projectService) checkPermission(ctx context.Context, claims *response.JWTClaims, projectUUID pgtype.UUID) *fiber.Error {
	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Project not found")
	}
	if project.UserID == claims.ID {
		return nil
	}

	for _, r := range claims.Roles {
		if r == constants.RoleTypeAdmin || r == constants.RoleTypeMod {
			return nil
		}
	}

	callerUUID, _ := convert.StringToUUID(claims.ID)
	role, err := s.projectRepo.CheckPermission(ctx, sqlc.CheckProjectPermissionParams{
		ProjectID: projectUUID,
		UserID:    callerUUID,
	})
	if err != nil || constants.ParseProjectMemberRole(role) != constants.ProjectMemberRoleOwner {
		return fiber.NewError(fiber.StatusForbidden, "Only project owner can perform this action")
	}

	return nil
}

func (s *projectService) GetProjectByID(ctx context.Context, id string) (*response.ProjectResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Project not found")
	}

	return project.ToResponse(), nil
}

func (s *projectService) GetProjectByUserID(ctx context.Context, userID string, dto *request.GetProjectsByUserDto) ([]*response.ProjectResponse, *fiber.Error) {
	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
	}

	limit := int32(20)
	if dto.Limit > 0 {
		limit = dto.Limit
	}

	arg := sqlc.GetProjectsByUserIdParams{
		UserID: userUUID,
		Limit:  limit,
	}

	if dto.CursorID != "" {
		if cursorID, err := convert.StringToUUID(dto.CursorID); err == nil {
			arg.CursorID = cursorID
		}
	}

	projects, err := s.projectRepo.GetByUserID(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user projects")
	}

	return models.ProjectsEntityToResponse(projects), nil
}

func (s *projectService) fillSearchArgs(arg *sqlc.SearchProjectsParams, dto *request.SearchProjectDto) {
	if dto.Sort != "" {
		arg.Sort = dto.Sort
	} else {
		arg.Sort = "updated_at"
	}

	arg.Order = "desc"
	if dto.Order == "asc" {
		arg.Order = "asc"
	}

	if len(dto.Statuses) > 0 {
		for _, statusStr := range dto.Statuses {
			statusType := constants.ParseProjectStatusTypeText(statusStr)
			if statusType != constants.ProjectStatusTypeUnknow {
				arg.Statuses = append(arg.Statuses, statusType.Int16())
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

func (s *projectService) SearchProject(ctx context.Context, dto *request.SearchProjectDto) (*response.PaginatedResponse, *fiber.Error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit

	arg := sqlc.SearchProjectsParams{
		Limit:  int32(dto.Limit),
		Offset: int32(offset),
	}

	s.fillSearchArgs(&arg, dto)

	var rows []*models.ProjectEntity
	var totalRecords int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = s.projectRepo.Search(gCtx, arg)
		return err
	})

	g.Go(func() error {
		countArg := sqlc.CountProjectsParams{
			Statuses:    arg.Statuses,
			UserIds:     arg.UserIds,
			SearchText:  arg.SearchText,
			CreatedFrom: arg.CreatedFrom,
			CreatedTo:   arg.CreatedTo,
		}
		var err error
		totalRecords, err = s.projectRepo.Count(gCtx, countArg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search projects")
	}

	projects := models.ProjectsEntityToResponse(rows)

	return response.BuildPaginatedResponse(projects, totalRecords, dto.Page, dto.Limit), nil
}

func (s *projectService) CreateProject(ctx context.Context, userID string, dto *request.CreateProjectDto) (*response.ProjectResponse, *fiber.Error) {
	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
	}

	arg := sqlc.CreateProjectParams{
		Title:         dto.Title,
		Description:   convert.PtrToText(dto.Description),
		ProjectStatus: constants.ProjectStatusTypePrivate.Int16(),
		UserID:        userUUID,
	}

	if dto.Status != nil {
		statusType := constants.ParseProjectStatusTypeText(*dto.Status)
		if statusType == constants.ProjectStatusTypeUnknow {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid status type")
		}
		arg.ProjectStatus = statusType.Int16()
	}

	project, err := s.projectRepo.Create(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create project")
	}

	return project.ToResponse(), nil
}

func (s *projectService) UpdateProject(ctx context.Context, claims *response.JWTClaims, id string, dto *request.UpdateProjectDto) (*response.ProjectResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return nil, fErr
	}

	arg := sqlc.UpdateProjectParams{
		ID:          projectUUID,
		Title:       convert.PtrToText(dto.Title),
		Description: convert.PtrToText(dto.Description),
	}

	if dto.Status != nil {
		statusType := constants.ParseProjectStatusTypeText(*dto.Status)
		if statusType == constants.ProjectStatusTypeUnknow {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid status type")
		}
		arg.Status = pgtype.Int2{Int16: statusType.Int16(), Valid: true}
	}

	project, err := s.projectRepo.Update(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update project")
	}

	return project.ToResponse(), nil
}

func (s *projectService) DeleteProject(ctx context.Context, claims *response.JWTClaims, id string) *fiber.Error {
	projectUUID, err := convert.StringToUUID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return fErr
	}

	err = s.projectRepo.Delete(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete project")
	}

	return nil
}

func (s *projectService) AddMember(ctx context.Context, claims *response.JWTClaims, projectID string, dto *request.AddProjectMemberDto) (*response.ProjectResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return nil, fErr
	}

	memberUUID, err := convert.StringToUUID(dto.UserID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid member user ID format")
	}

	callerUUID, _ := convert.StringToUUID(claims.ID)

	if dto.UserID == claims.ID {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Cannot add yourself as a member")
	}

	role := constants.ParseProjectMemberRoleText(dto.Role)

	err = s.projectRepo.AddMember(ctx, sqlc.AddProjectMemberParams{
		ProjectID: projectUUID,
		UserID:    memberUUID,
		Role:      role.Int16(),
		InvitedBy: callerUUID,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusConflict, "Member already exists or user not found")
	}

	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch updated project")
	}

	return project.ToResponse(), nil
}

func (s *projectService) UpdateMemberRole(ctx context.Context, claims *response.JWTClaims, projectID string, memberUserID string, dto *request.UpdateProjectMemberDto) (*response.ProjectResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return nil, fErr
	}

	memberUUID, err := convert.StringToUUID(memberUserID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid member user ID format")
	}

	role := constants.ParseProjectMemberRoleText(dto.Role)

	err = s.projectRepo.UpdateMemberRole(ctx, sqlc.UpdateProjectMemberRoleParams{
		ProjectID: projectUUID,
		UserID:    memberUUID,
		Role:      role.Int16(),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Member not found")
	}

	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch updated project")
	}

	return project.ToResponse(), nil
}

func (s *projectService) RemoveMember(ctx context.Context, claims *response.JWTClaims, projectID string, memberUserID string) *fiber.Error {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return fErr
	}

	memberUUID, err := convert.StringToUUID(memberUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid member user ID format")
	}

	if claims.ID == memberUserID {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot remove yourself from the project")
	}

	err = s.projectRepo.RemoveMember(ctx, sqlc.RemoveProjectMemberParams{
		ProjectID: projectUUID,
		UserID:    memberUUID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Member not found")
	}

	return nil
}

func (s *projectService) ChangeOwner(ctx context.Context, claims *response.JWTClaims, projectID string, newOwnerID string) (*response.ProjectResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(projectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID format")
	}

	if fErr := s.checkPermission(ctx, claims, projectUUID); fErr != nil {
		return nil, fErr
	}

	if claims.ID == newOwnerID {
		return nil, fiber.NewError(fiber.StatusBadRequest, "You are already the owner")
	}

	newOwnerUUID, err := convert.StringToUUID(newOwnerID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid new owner ID format")
	}

	err = s.projectRepo.ChangeOwner(ctx, sqlc.ChangeProjectOwnerParams{
		ID:     projectUUID,
		UserID: newOwnerUUID,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to change owner")
	}

	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch updated project")
	}

	return project.ToResponse(), nil
}

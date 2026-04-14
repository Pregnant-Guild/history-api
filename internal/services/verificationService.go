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
	"golang.org/x/sync/errgroup"
)

type VerificationService interface {
	GetVerificationByID(ctx context.Context, verificationId string) (*response.UserVerificationResponse, error)
	GetVerificationByUserID(ctx context.Context, userId string) ([]*response.UserVerificationResponse, error)
	SearchVerification(ctx context.Context, dto *request.SearchUserVerificationDto) (*response.PaginatedResponse, error)
	DeleteVerification(ctx context.Context, claims *response.JWTClaims, verificationId string) error
	CreateVerification(ctx context.Context, userId string, dto *request.CreateUserVerificationDto) (*response.UserVerificationResponse, error)
	UpdateStatusVerification(ctx context.Context, userId string, verificationId string, dto *request.UpdateVerificationStatusDto) (*response.UserVerificationResponse, error)
}

type verificationService struct {
	verificationRepo repositories.VerificationRepository
	mediaRepo        repositories.MediaRepository
	userRepo         repositories.UserRepository
	roleRepo         repositories.RoleRepository
	c                cache.Cache
}

func NewVerificationService(
	verificationRepo repositories.VerificationRepository,
	mediaRepo repositories.MediaRepository,
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	c cache.Cache,
) VerificationService {
	return &verificationService{
		verificationRepo: verificationRepo,
		mediaRepo:        mediaRepo,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		c:                c,
	}
}

func (v *verificationService) CreateVerification(ctx context.Context, userId string, dto *request.CreateUserVerificationDto) (*response.UserVerificationResponse, error) {
	verifyType := constants.ParseVerifyTypeText(dto.VerifyType)
	if verifyType == constants.VerifyUnknown {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Unknown verify type!")
	}

	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	mediaList, err := v.mediaRepo.GetByIDs(ctx, dto.MediaIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if len(mediaList) != len(dto.MediaIDs) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Some media IDs are invalid!")
	}

	item, err := v.verificationRepo.Create(
		ctx,
		sqlc.CreateUserVerificationParams{
			VerifyType: verifyType.Int16(),
			Content:    convert.PtrToText(&dto.Content),
			UserID:     pgID,
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	itemId, err := convert.StringToUUID(item.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	mediaIdList := make([]pgtype.UUID, 0)
	for _, it := range mediaList {
		mediaId, err := convert.StringToUUID(it.ID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		mediaIdList = append(mediaIdList, mediaId)
		item.Media = append(item.Media, it.ToSimpleEntity())
	}

	err = v.verificationRepo.CreateVerificationMedia(
		ctx,
		sqlc.CreateVerificationMediaParams{
			VerificationID: itemId,
			Column2:        mediaIdList,
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return item.ToResponse(), nil
}

func (v *verificationService) DeleteVerification(ctx context.Context, claims *response.JWTClaims, verificationId string) error {
	verificationIdUUID, err := convert.StringToUUID(verificationId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	verification, err := v.verificationRepo.GetByID(ctx, verificationIdUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	shoudDelete := false
	if slices.Contains(claims.Roles, constants.ADMIN) || slices.Contains(claims.Roles, constants.MOD) || verification.UserID == claims.UId {
		shoudDelete = true
	}

	if !shoudDelete {
		return fiber.NewError(fiber.StatusForbidden, "You don't have permission to delete this verification")
	}

	err = v.mediaRepo.Delete(ctx, verificationIdUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return nil
}

func (v *verificationService) GetVerificationByID(ctx context.Context, verificationId string) (*response.UserVerificationResponse, error) {
	verificationUUID, err := convert.StringToUUID(verificationId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	verification, err := v.verificationRepo.GetByID(ctx, verificationUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return verification.ToResponse(), nil
}

func (v *verificationService) GetVerificationByUserID(ctx context.Context, userId string) ([]*response.UserVerificationResponse, error) {
	userUUID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	verifications, err := v.verificationRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return models.UserVerificationsEntitiesToResponse(verifications), nil
}

func (m *verificationService) fillSearchArgs(arg *sqlc.SearchUserVerificationsParams, dto *request.SearchUserVerificationDto) {
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
			if u := constants.ParseStatusTypeText(id); u == constants.StatusUnknown {
				arg.Statuses = append(arg.Statuses, u.Int16())
			}
		}
	}

	if len(dto.VerifyTypes) > 0 {
		for _, id := range dto.VerifyTypes {
			if u := constants.ParseVerifyTypeText(id); u == constants.VerifyUnknown {
				arg.VerifyTypes = append(arg.VerifyTypes, u.Int16())
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

func (v *verificationService) SearchVerification(ctx context.Context, dto *request.SearchUserVerificationDto) (*response.PaginatedResponse, error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit

	arg := sqlc.SearchUserVerificationsParams{
		Limit:  int32(dto.Limit),
		Offset: int32(offset),
	}

	v.fillSearchArgs(&arg, dto)

	var rows []*models.UserVerificationEntity
	var totalRecords int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = v.verificationRepo.Search(gCtx, arg)
		return err
	})

	g.Go(func() error {
		countArg := sqlc.CountUserVerificationsParams{
			UserIds:     arg.UserIds,
			Statuses:    arg.Statuses,
			VerifyTypes: arg.VerifyTypes,
			ReviewedBy:  arg.ReviewedBy,
			CreatedFrom: arg.CreatedFrom,
			CreatedTo:   arg.CreatedTo,
			SearchText:  arg.SearchText,
		}
		var err error
		totalRecords, err = v.verificationRepo.Count(gCtx, countArg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	verifications := models.UserVerificationsEntitiesToResponse(rows)

	return response.BuildPaginatedResponse(verifications, totalRecords, dto.Page, dto.Limit), nil
}

func (v *verificationService) UpdateStatusVerification(ctx context.Context, userId string, verificationId string, dto *request.UpdateVerificationStatusDto) (*response.UserVerificationResponse, error) {
	statusType := constants.ParseStatusTypeText(dto.Status)
	if statusType == constants.StatusUnknown {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Unknown status type!")
	}
	verificationUUID, err := convert.StringToUUID(verificationId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	userAdminUUID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	historianRole, err := v.roleRepo.GetByname(ctx, constants.HISTORIAN.String())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	historianRoleID, err := convert.StringToUUID(historianRole.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	verification, err := v.verificationRepo.GetByID(ctx, verificationUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if verification.Status != constants.StatusPending {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid status!")
	}

	userVerificationUUID, err := convert.StringToUUID(verification.UserID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	userVerification, err := v.userRepo.GetByID(ctx, userVerificationUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = v.verificationRepo.UpdateStatus(
		ctx,
		sqlc.UpdateUserVerificationStatusParams{
			ID:         verificationUUID,
			Status:     statusType.Int16(),
			ReviewedBy: userAdminUUID,
			ReviewNote: convert.PtrToText(&dto.ReviewNote),
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	verification.Status = statusType

	data := &models.UserVerificationStorageEntity{
		Email:      userVerification.Email,
		Name:       userVerification.Profile.DisplayName,
		ReviewNote: dto.ReviewNote,
		Status:     statusType,
	}

	if statusType == constants.StatusApproved {
		roleIdList := make([]pgtype.UUID, 0)
		userVerification.Roles = append(userVerification.Roles, historianRole.ToRoleSimple())

		roleIdList = append(roleIdList, historianRoleID)

		for _, role := range userVerification.Roles {
			roleID, err := convert.StringToUUID(role.ID)
			if err != nil {
				continue
			}
			roleIdList = append(roleIdList, roleID)
		}

		err = v.roleRepo.BulkDeleteRolesFromUser(ctx, userVerificationUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		err = v.roleRepo.CreateUserRole(ctx, sqlc.CreateUserRoleParams{
			UserID:  userVerificationUUID,
			Column2: roleIdList,
		})
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		err = v.userRepo.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
			ID:           userVerificationUUID,
			TokenVersion: userVerification.TokenVersion + 1,
		})
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		userVerification.TokenVersion += 1

		mapCache := map[string]any{
			fmt.Sprintf("user:email:%s", userVerification.Email): userVerification,
			fmt.Sprintf("user:id:%s", userVerification.ID):       userVerification,
		}
		_ = v.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
	}

	v.c.PublishTask(ctx, constants.StreamEmailName, constants.TaskTypeNotifyHistorianReview, data)

	return verification.ToResponse(), nil
}

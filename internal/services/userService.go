package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

type UserService interface {
	//user
	GetUserCurrent(ctx context.Context, userId string) (*response.UserResponse, error)
	UpdateProfile(ctx context.Context, userId string, dto *request.UpdateProfileDto) (*response.UserResponse, error)
	ChangePassword(ctx context.Context, userId string, dto *request.ChangePasswordDto) error

	//admin
	DeleteUser(ctx context.Context, userId string) error
	ChangeRoleUser(ctx context.Context, dto *request.ChangeRoleDto) (*response.UserResponse, error)
	RestoreUser(ctx context.Context, userId string) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userId string) (*response.UserResponse, error)
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error)
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
}

func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
) UserService {
	return &userService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (u *userService) ChangePassword(ctx context.Context, userId string, dto *request.ChangePasswordDto) error {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.OldPassword)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid identity or password!")
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = u.userRepo.UpdatePassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           pgID,
		PasswordHash: pgtype.Text{String: string(hashPassword), Valid: true},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return nil
}

func (u *userService) ChangeRoleUser(ctx context.Context, dto *request.ChangeRoleDto) (*response.UserResponse, error) {
	userId, err := convert.StringToUUID(dto.UserID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	user, err := u.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	roleIdstr, err := u.roleRepo.GetByIDs(ctx, dto.Roles)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user.Roles = make([]*models.RoleSimple, 0)
	roleIdList := make([]pgtype.UUID, 0)
	for _, role := range roleIdstr {
		roleID, err := convert.StringToUUID(role.ID)
		if err != nil {
			continue
		}
		roleIdList = append(roleIdList, roleID)
		user.Roles = append(user.Roles, role.ToRoleSimple())
	}

	err = u.roleRepo.RemoveAllRolesFromUser(ctx, userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = u.roleRepo.AddUserRole(ctx, sqlc.AddUserRoleParams{
		UserID:  userId,
		Column2: roleIdList,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return user.ToResponse(), nil

}

func (u *userService) DeleteUser(ctx context.Context, userId string) error {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	err = u.userRepo.Delete(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return nil
}

func (u *userService) UpdateProfile(ctx context.Context, userId string, dto *request.UpdateProfileDto) (*response.UserResponse, error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	newUser, err := u.userRepo.UpdateProfile(
		ctx,
		sqlc.UpdateUserProfileParams{
			DisplayName: pgtype.Text{String: dto.DisplayName, Valid: len(dto.DisplayName) > 0},
			FullName:    pgtype.Text{String: dto.FullName, Valid: len(dto.FullName) > 0},
			AvatarUrl:   pgtype.Text{String: dto.AvatarUrl, Valid: len(dto.AvatarUrl) > 0},
			Bio:         pgtype.Text{String: dto.Bio, Valid: len(dto.Bio) > 0},
			Location:    pgtype.Text{String: dto.Location, Valid: len(dto.Location) > 0},
			Website:     pgtype.Text{String: dto.Website, Valid: len(dto.Website) > 0},
			CountryCode: pgtype.Text{String: dto.CountryCode, Valid: len(dto.CountryCode) > 0},
			Phone:       pgtype.Text{String: dto.Phone, Valid: len(dto.Phone) > 0},
			UserID:      pgID,
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return newUser.ToResponse(), nil
}

func (u *userService) GetUserCurrent(ctx context.Context, userId string) (*response.UserResponse, error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return user.ToResponse(), nil
}

func (u *userService) RestoreUser(ctx context.Context, userId string) (*response.UserResponse, error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	err = u.userRepo.Restore(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user.IsDeleted = false
	return user.ToResponse(), nil
}

func (m *userService) fillSearchArgs(arg *sqlc.SearchUsersParams, dto *request.SearchUserDto) {
	if dto.Sort != "" {
		arg.Sort = pgtype.Text{String: dto.Sort, Valid: true}
	} else {
		arg.Sort = pgtype.Text{String: "id", Valid: true}
	}

	arg.Order = pgtype.Text{String: "asc", Valid: true}
	if dto.Order == "desc" {
		arg.Order = pgtype.Text{String: "desc", Valid: true}
	}

	if dto.AuthProvider != "" {
		arg.AuthProvider = pgtype.Text{String: dto.AuthProvider, Valid: true}
	}

	if dto.CreatedFrom != nil {
		arg.CreatedFrom = pgtype.Timestamp{Time: *dto.CreatedFrom, Valid: true}
	}

	if dto.CreatedTo != nil {
		arg.CreatedTo = pgtype.Timestamp{Time: *dto.CreatedTo, Valid: true}
	}

	if dto.IsDeleted != nil {
		arg.IsDeleted = pgtype.Bool{Bool: *dto.IsDeleted, Valid: true}
	}

	if len(dto.RoleIDs) > 0 {
		for _, id := range dto.RoleIDs {
			if u, err := convert.StringToUUID(id); err == nil {
				arg.RoleIds = append(arg.RoleIds, u)
			}
		}
	}

	if dto.Search != "" {
		arg.SearchText = pgtype.Text{String: dto.Search, Valid: true}
	}
}

func (u *userService) SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit

	arg := sqlc.SearchUsersParams{
		Limit:  int32(dto.Limit),
		Offset: int32(offset),
	}

	u.fillSearchArgs(&arg, dto)

	var rows []*models.UserEntity
	var totalRecords int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = u.userRepo.Search(gCtx, arg)
		return err
	})

	g.Go(func() error {
		countArg := sqlc.CountUsersParams{
			RoleIds:      arg.RoleIds,
			AuthProvider: arg.AuthProvider,
			CreatedFrom:  arg.CreatedFrom,
			CreatedTo:    arg.CreatedTo,
			IsDeleted:    arg.IsDeleted,
			SearchText:   arg.SearchText,
		}
		var err error
		totalRecords, err = u.userRepo.Count(gCtx, countArg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return response.BuildPaginatedResponse(rows, totalRecords, dto.Page, dto.Limit), nil
}

func (u *userService) GetUserByID(ctx context.Context, userId string) (*response.UserResponse, error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return user.ToResponse(), nil
}

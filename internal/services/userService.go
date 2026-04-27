package services

import (
	"context"
	"database/sql"
	"errors"
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
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

type UserService interface {
	//user
	UpdateProfile(ctx context.Context, userId string, dto *request.UpdateProfileDto) (*response.UserResponse, *fiber.Error)
	ChangePassword(ctx context.Context, userId string, dto *request.ChangePasswordDto) *fiber.Error

	//admin
	CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, *fiber.Error)
	DeleteUser(ctx context.Context, userId string) *fiber.Error
	ChangeRoleUser(ctx context.Context, userId string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, *fiber.Error)
	RestoreUser(ctx context.Context, userId string) (*response.UserResponse, *fiber.Error)
	GetUserByID(ctx context.Context, userId string) (*response.UserResponse, *fiber.Error)
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, *fiber.Error)
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	c        cache.Cache
	db       *pgxpool.Pool
}

func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	c cache.Cache,
	db *pgxpool.Pool,
) UserService {
	return &userService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		c:        c,
		db:       db,
	}
}

func (u *userService) CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, *fiber.Error) {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepo := u.userRepo.WithTx(tx)
	rRepo := u.roleRepo.WithTx(tx)

	existingUser, err := u.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to check existing user")
	}
	if existingUser != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "User already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}

	user, err := uRepo.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email:        dto.Email,
		PasswordHash: pgtype.Text{String: string(hashed), Valid: true},
		AuthProvider: constants.ProviderTypeLocal.String(),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user")
	}

	userUUID, err := convert.StringToUUID(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}

	_, err = uRepo.CreateProfile(ctx, sqlc.CreateUserProfileParams{
		UserID:      userUUID,
		DisplayName: pgtype.Text{String: dto.DisplayName, Valid: true},
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user profile")
	}

	var roleIdList []pgtype.UUID
	for _, rId := range dto.Roles {
		rid, err := convert.StringToUUID(rId)
		if err == nil {
			roleIdList = append(roleIdList, rid)
		}
	}

	err = rRepo.CreateUserRole(ctx, sqlc.CreateUserRoleParams{
		UserID:  userUUID,
		Column2: roleIdList,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign roles")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction")
	}
	
	finalUser, err := u.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch created user")
	}

	return finalUser.ToResponse(), nil
}

func (u *userService) ChangePassword(ctx context.Context, userId string, dto *request.ChangePasswordDto) *fiber.Error {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepo := u.userRepo.WithTx(tx)

	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if user.PasswordHash != "" {
		if dto.OldPassword == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Old password required")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.OldPassword)); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid old password")
		}
	} else if user.PasswordHash == "" && dto.OldPassword != "" {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request: user has no password")
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash new password")
	}

	err = uRepo.UpdatePassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           pgID,
		PasswordHash: pgtype.Text{String: string(hashPassword), Valid: true},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update password")
	}

	err = uRepo.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
		ID:           pgID,
		TokenVersion: user.TokenVersion + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update token version")
	}

	if err := tx.Commit(ctx); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction")
	}
	return nil
}

func (u *userService) ChangeRoleUser(ctx context.Context, userId string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, *fiber.Error) {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepo := u.userRepo.WithTx(tx)
	rRepo := u.roleRepo.WithTx(tx)

	userUUID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}

	user, err := u.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	newListRole, err := u.roleRepo.GetByIDs(ctx, dto.Roles)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch roles")
	}

	hasUserRole := false
	hasAdminRole := false
	hasBannedRole := false
	hasModRole := false

	for _, r := range newListRole {
		if r.Name == constants.RoleTypeUser.String() {
			hasUserRole = true
		}
		if r.Name == constants.RoleTypeAdmin.String() {
			hasAdminRole = true
		}
		if r.Name == constants.RoleTypeBanned.String() {
			hasBannedRole = true
		}
		if r.Name == constants.RoleTypeMod.String() {
			hasModRole = true
		}
	}

	if !hasUserRole {
		return nil, fiber.NewError(fiber.StatusForbidden, "User must have the USER role")
	}

	if slices.Contains(claims.Roles, constants.RoleTypeMod) && !slices.Contains(claims.Roles, constants.RoleTypeAdmin) {
		if hasAdminRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "MOD cannot assign ADMIN role")
		}

		if userId == claims.UId && !hasModRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "You cannot remove your own MOD role")
		}

		if userId == claims.UId && hasBannedRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "You cannot ban yourself")
		}
		isTargetAdminOrMod := false
		for _, r := range user.Roles {
			if r.Name == constants.RoleTypeAdmin.String() || r.Name == constants.RoleTypeMod.String() {
				isTargetAdminOrMod = true
				break
			}
		}
		if isTargetAdminOrMod && hasBannedRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "MOD cannot ban an ADMIN or MOD")
		}
	}

	if slices.Contains(claims.Roles, constants.RoleTypeAdmin) {
		if userId == claims.UId && hasBannedRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "You cannot ban yourself")
		}

		if userId == claims.UId && !hasAdminRole {
			return nil, fiber.NewError(fiber.StatusForbidden, "You cannot remove your own ADMIN role")
		}
	}

	user.Roles = make([]*models.RoleSimple, 0)
	roleIdList := make([]pgtype.UUID, 0)
	for _, role := range newListRole {
		roleID, err := convert.StringToUUID(role.ID)
		if err != nil {
			continue
		}
		roleIdList = append(roleIdList, roleID)
		user.Roles = append(user.Roles, role.ToRoleSimple())
	}

	err = rRepo.BulkDeleteRolesFromUser(ctx, userUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to clear old roles")
	}

	err = rRepo.CreateUserRole(ctx, sqlc.CreateUserRoleParams{
		UserID:  userUUID,
		Column2: roleIdList,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create new roles")
	}

	err = uRepo.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
		ID:           userUUID,
		TokenVersion: user.TokenVersion + 1,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update token version")
	}
	user.TokenVersion += 1

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction")
	}

	mapCache := map[string]any{
		fmt.Sprintf("user:email:%s", user.Email): user,
		fmt.Sprintf("user:id:%s", user.ID):       user,
	}
	_ = u.c.MSet(ctx, mapCache, constants.NormalCacheDuration)

	return user.ToResponse(), nil
}

func (u *userService) DeleteUser(ctx context.Context, userId string) *fiber.Error {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	err = u.userRepo.Delete(ctx, pgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete user")
	}
	return nil
}

func (u *userService) UpdateProfile(ctx context.Context, userId string, dto *request.UpdateProfileDto) (*response.UserResponse, *fiber.Error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	newUser, err := u.userRepo.UpdateProfile(
		ctx,
		sqlc.UpdateUserProfileParams{
			DisplayName: convert.PtrToText(dto.DisplayName),
			FullName:    convert.PtrToText(dto.FullName),
			AvatarUrl:   convert.PtrToText(dto.AvatarUrl),
			Bio:         convert.PtrToText(dto.Bio),
			Location:    convert.PtrToText(dto.Location),
			Website:     convert.PtrToText(dto.Website),
			CountryCode: convert.PtrToText(dto.CountryCode),
			Phone:       convert.PtrToText(dto.Phone),
			UserID:      pgID,
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update profile")
	}
	return newUser.ToResponse(), nil
}

func (u *userService) RestoreUser(ctx context.Context, userId string) (*response.UserResponse, *fiber.Error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}

	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	err = u.userRepo.Restore(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to restore user")
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
		arg.CreatedFrom = pgtype.Timestamptz{Time: *dto.CreatedFrom, Valid: true}
	}

	if dto.CreatedTo != nil {
		arg.CreatedTo = pgtype.Timestamptz{Time: *dto.CreatedTo, Valid: true}
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

func (u *userService) SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, *fiber.Error) {
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search users")
	}

	users := models.UsersEntityToResponse(rows)

	return response.BuildPaginatedResponse(users, totalRecords, dto.Page, dto.Limit), nil
}

func (u *userService) GetUserByID(ctx context.Context, userId string) (*response.UserResponse, *fiber.Error) {
	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}
	user, err := u.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Failed to fetch user")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return user.ToResponse(), nil
}

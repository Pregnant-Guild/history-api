package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/convert"
)

type UserRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	All(ctx context.Context) ([]*models.UserEntity, error)
	UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error)
	CreateProfile(ctx context.Context, params sqlc.CreateUserProfileParams) (*models.UserProfileSimple, error)
	UpdateProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error
	UpdateRefreshToken(ctx context.Context, params sqlc.UpdateUserRefreshTokenParams) error
	GetTokenVersion(ctx context.Context, id pgtype.UUID) (int32, error)
	UpdateTokenVersion(ctx context.Context, params sqlc.UpdateTokenVersionParams) error
	Verify(ctx context.Context, id pgtype.UUID) error
	Delete(ctx context.Context, id pgtype.UUID) error
	Restore(ctx context.Context, id pgtype.UUID) error
}

type userRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewUserRepository(db sqlc.DBTX, c cache.Cache) UserRepository {
	return &userRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *userRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error) {
	cacheId := fmt.Sprintf("user:id:%s", convert.UUIDToString(id))
	var user models.UserEntity
	err := r.c.Get(ctx, cacheId, &user)
	if err == nil {
		return &user, nil
	}

	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user = models.UserEntity{
		ID:           convert.UUIDToString(row.ID),
		Email:        row.Email,
		PasswordHash: convert.TextToString(row.PasswordHash),
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}

	if err := user.ParseRoles(row.Roles); err != nil {
		return nil, err
	}

	if err := user.ParseProfile(row.Profile); err != nil {
		return nil, err
	}

	_ = r.c.Set(ctx, cacheId, user, 5*time.Minute)

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	cacheId := fmt.Sprintf("user:email:%s", email)

	var user models.UserEntity
	err := r.c.Get(ctx, cacheId, &user)
	if err == nil {
		return &user, nil
	}

	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	user = models.UserEntity{
		ID:           convert.UUIDToString(row.ID),
		Email:        row.Email,
		PasswordHash: convert.TextToString(row.PasswordHash),
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}

	if err := user.ParseRoles(row.Roles); err != nil {
		return nil, err
	}

	if err := user.ParseProfile(row.Profile); err != nil {
		return nil, err
	}

	_ = r.c.Set(ctx, cacheId, user, 5*time.Minute)

	return &user, nil
}

func (r *userRepository) UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error) {
	row, err := r.q.UpsertUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return &models.UserEntity{
		ID:           convert.UUIDToString(row.ID),
		Email:        row.Email,
		PasswordHash: convert.TextToString(row.PasswordHash),
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
		Roles:        make([]*models.RoleSimple, 0),
	}, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (*models.UserEntity, error) {
	user, err := r.GetByID(ctx, params.UserID)
	if err != nil {
		return nil, err
	}

	row, err := r.q.UpdateUserProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	profile := models.UserProfileSimple{
		DisplayName: convert.TextToString(row.DisplayName),
		FullName:    convert.TextToString(row.FullName),
		AvatarUrl:   convert.TextToString(row.AvatarUrl),
		Bio:         convert.TextToString(row.Bio),
		Location:    convert.TextToString(row.Location),
		Website:     convert.TextToString(row.Website),
		CountryCode: convert.TextToString(row.CountryCode),
		Phone:       convert.TextToString(row.Phone),
	}

	user.Profile = &profile
	mapCache := map[string]any{
		fmt.Sprintf("user:email:%s", user.Email): user,
		fmt.Sprintf("user:id:%s", user.ID):       user,
	}
	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
	return user, nil
}

func (r *userRepository) CreateProfile(ctx context.Context, params sqlc.CreateUserProfileParams) (*models.UserProfileSimple, error) {
	row, err := r.q.CreateUserProfile(ctx, params)
	if err != nil {
		return nil, err
	}

	return &models.UserProfileSimple{
		DisplayName: convert.TextToString(row.DisplayName),
		FullName:    convert.TextToString(row.FullName),
		AvatarUrl:   convert.TextToString(row.AvatarUrl),
		Bio:         convert.TextToString(row.Bio),
		Location:    convert.TextToString(row.Location),
		Website:     convert.TextToString(row.Website),
		CountryCode: convert.TextToString(row.CountryCode),
		Phone:       convert.TextToString(row.Phone),
	}, nil
}

func (r *userRepository) All(ctx context.Context) ([]*models.UserEntity, error) {
	rows, err := r.q.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	var users []*models.UserEntity
	for _, row := range rows {
		user := &models.UserEntity{
			ID:           convert.UUIDToString(row.ID),
			Email:        row.Email,
			PasswordHash: convert.TextToString(row.PasswordHash),
			IsVerified:   row.IsVerified,
			TokenVersion: row.TokenVersion,
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
		}

		if err := user.ParseRoles(row.Roles); err != nil {
			return nil, err
		}

		if err := user.ParseProfile(row.Profile); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) Verify(ctx context.Context, id pgtype.UUID) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = r.q.VerifyUser(ctx, id)
	if err != nil {
		return err
	}
	err = r.q.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
		ID:           id,
		TokenVersion: user.TokenVersion + 1,
	})
	if err != nil {
		return err
	}

	user.IsVerified = true
	user.TokenVersion += 1

	mapCache := map[string]any{
		fmt.Sprintf("user:email:%s", user.Email): user,
		fmt.Sprintf("user:id:%s", user.ID):       user,
	}
	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = r.q.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	_ = r.c.Del(
		ctx,
		fmt.Sprintf("user:id:%s", user.ID),
		fmt.Sprintf("user:email:%s", user.Email),
		fmt.Sprintf("user:token:%s", user.ID),
	)
	return nil
}

func (r *userRepository) Restore(ctx context.Context, id pgtype.UUID) error {
	err := r.q.RestoreUser(ctx, id)
	if err != nil {
		return err
	}

	_ = r.c.Del(ctx, fmt.Sprintf("user:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *userRepository) GetTokenVersion(ctx context.Context, id pgtype.UUID) (int32, error) {
	cacheId := fmt.Sprintf("user:token:%s", convert.UUIDToString(id))
	var token int32
	err := r.c.Get(ctx, cacheId, &token)
	if err == nil {
		return token, nil
	}

	raw, err := r.q.GetTokenVersion(ctx, id)
	if err != nil {
		return 0, err
	}

	_ = r.c.Set(ctx, cacheId, raw, 5*time.Minute)
	return raw, nil
}

func (r *userRepository) UpdateTokenVersion(ctx context.Context, params sqlc.UpdateTokenVersionParams) error {
	err := r.q.UpdateTokenVersion(ctx, params)
	if err != nil {
		return err
	}

	cacheId := fmt.Sprintf("user:token:%s", convert.UUIDToString(params.ID))
	_ = r.c.Set(ctx, cacheId, params.TokenVersion, 5*time.Minute)
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error {
	user, err := r.GetByID(ctx, params.ID)
	if err != nil {
		return err
	}
	err = r.q.UpdateUserPassword(ctx, params)
	if err != nil {
		return err
	}
	err = r.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
		ID:           params.ID,
		TokenVersion: user.TokenVersion + 1,
	})
	if err != nil {
		return err
	}

	user.PasswordHash = convert.TextToString(params.PasswordHash)
	user.TokenVersion += 1
	mapCache := map[string]any{
		fmt.Sprintf("user:email:%s", user.Email): user,
		fmt.Sprintf("user:id:%s", user.ID):       user,
		fmt.Sprintf("user:token:%s", user.ID):    user.TokenVersion,
	}

	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
	return nil
}

func (r *userRepository) UpdateRefreshToken(ctx context.Context, params sqlc.UpdateUserRefreshTokenParams) error {
	user, err := r.GetByID(ctx, params.ID)
	if err != nil {
		return err
	}
	err = r.q.UpdateUserRefreshToken(ctx, params)
	if err != nil {
		return err
	}

	user.RefreshToken = convert.TextToString(params.RefreshToken)
	mapCache := map[string]any{
		fmt.Sprintf("user:email:%s", user.Email): user,
		fmt.Sprintf("user:id:%s", user.ID):       user,
		fmt.Sprintf("user:token:%s", user.ID):    user.TokenVersion,
	}

	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
	return nil
}

package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
)

type UserRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error)
	GetByIDWithoutDeleted(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	Search(ctx context.Context, params sqlc.SearchUsersParams) ([]*models.UserEntity, error)
	Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error)
	UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error)
	CreateProfile(ctx context.Context, params sqlc.CreateUserProfileParams) (*models.UserProfileSimple, error)
	UpdateProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error
	UpdateRefreshToken(ctx context.Context, params sqlc.UpdateUserRefreshTokenParams) error
	GetTokenVersion(ctx context.Context, id pgtype.UUID) (int32, error)
	UpdateTokenVersion(ctx context.Context, params sqlc.UpdateTokenVersionParams) error
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

func (r *userRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *userRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.UserEntity, error) {
	if len(ids) == 0 {
		return []*models.UserEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("user:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var users []*models.UserEntity
	missingUsersToCache := make(map[string]any)

	for i, b := range raws {
		if len(b) > 0 {
			var u models.UserEntity
			if err := json.Unmarshal(b, &u); err == nil {
				users = append(users, &u)
			}
		} else {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err != nil {
				continue
			}
			dbUser, err := r.GetByID(ctx, pgId)
			if err == nil && dbUser != nil {
				users = append(users, dbUser)
				missingUsersToCache[keys[i]] = dbUser
			}
		}
	}

	if len(missingUsersToCache) > 0 {
		_ = r.c.MSet(ctx, missingUsersToCache, constants.NormalCacheDuration)
	}

	return users, nil
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

	_ = r.c.Set(ctx, cacheId, user, constants.NormalCacheDuration)

	return &user, nil
}

func (r *userRepository) GetByIDWithoutDeleted(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error) {
	cacheId := fmt.Sprintf("user:deleted:id:%s", convert.UUIDToString(id))
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

	_ = r.c.Set(ctx, cacheId, user, constants.NormalCacheDuration)

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

	_ = r.c.Set(ctx, cacheId, user, constants.NormalCacheDuration)

	return &user, nil
}

func (r *userRepository) UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error) {
	row, err := r.q.UpsertUser(ctx, params)
	if err != nil {
		return nil, err
	}
	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "user:search*")
		_ = r.c.DelByPattern(bgCtx, "user:count*")
	}()

	return &models.UserEntity{
		ID:           convert.UUIDToString(row.ID),
		Email:        row.Email,
		PasswordHash: convert.TextToString(row.PasswordHash),
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
	_ = r.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
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

func (r *userRepository) Search(ctx context.Context, params sqlc.SearchUsersParams) ([]*models.UserEntity, error) {
	queryKey := r.generateQueryKey("user:search", params)

	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchUsers(ctx, params)
	if err != nil {
		return nil, err
	}

	var users []*models.UserEntity
	var ids []string
	usersToCache := make(map[string]any)

	for _, row := range rows {
		user := &models.UserEntity{
			ID:           convert.UUIDToString(row.ID),
			Email:        row.Email,
			PasswordHash: convert.TextToString(row.PasswordHash),
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
		ids = append(ids, user.ID)
		usersToCache[fmt.Sprintf("user:id:%s", user.ID)] = user
	}

	if len(usersToCache) > 0 {
		_ = r.c.MSet(ctx, usersToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return users, nil
}

func (r *userRepository) Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error) {
	queryKey := r.generateQueryKey("user:count", params)
	var count int64
	if err := r.c.Get(ctx, queryKey, &count); err == nil {
		return count, nil
	}

	count, err := r.q.CountUsers(ctx, params)
	if err != nil {
		return 0, err
	}

	_ = r.c.Set(ctx, queryKey, count, constants.NormalCacheDuration)
	return count, nil
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

	_ = r.c.Set(ctx, cacheId, raw, constants.NormalCacheDuration)
	return raw, nil
}

func (r *userRepository) UpdateTokenVersion(ctx context.Context, params sqlc.UpdateTokenVersionParams) error {
	err := r.q.UpdateTokenVersion(ctx, params)
	if err != nil {
		return err
	}

	cacheId := fmt.Sprintf("user:token:%s", convert.UUIDToString(params.ID))
	_ = r.c.Set(ctx, cacheId, params.TokenVersion, constants.NormalCacheDuration)
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

	_ = r.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
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

	_ = r.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
	return nil
}

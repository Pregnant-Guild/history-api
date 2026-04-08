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

type RoleRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.RoleEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.RoleEntity, error)
	GetByname(ctx context.Context, name string) (*models.RoleEntity, error)
	All(ctx context.Context) ([]*models.RoleEntity, error)
	Create(ctx context.Context, name string) (*models.RoleEntity, error)
	Update(ctx context.Context, params sqlc.UpdateRoleParams) (*models.RoleEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	Restore(ctx context.Context, id pgtype.UUID) error
	AddUserRole(ctx context.Context, params sqlc.AddUserRoleParams) error
	RemoveUserRole(ctx context.Context, params sqlc.RemoveUserRoleParams) error
	RemoveAllRolesFromUser(ctx context.Context, userId pgtype.UUID) error
	RemoveAllUsersFromRole(ctx context.Context, roleId pgtype.UUID) error
}

type roleRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewRoleRepository(db sqlc.DBTX, c cache.Cache) RoleRepository {
	return &roleRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *roleRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *roleRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.RoleEntity, error) {
	if len(ids) == 0 {
		return []*models.RoleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("role:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var roles []*models.RoleEntity
	missingRolesToCache := make(map[string]any)

	for i, b := range raws {
		if len(b) > 0 {
			var u models.RoleEntity
			if err := json.Unmarshal(b, &u); err == nil {
				roles = append(roles, &u)
			}
		} else {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err != nil {
				continue
			}
			dbRole, err := r.GetByID(ctx, pgId)
			if err == nil && dbRole != nil {
				roles = append(roles, dbRole)
				missingRolesToCache[keys[i]] = dbRole
			}
		}
	}

	if len(missingRolesToCache) > 0 {
		_ = r.c.MSet(ctx, missingRolesToCache, constants.NormalCacheDuration)
	}

	return roles, nil
}

func (r *roleRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.RoleEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *roleRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.RoleEntity, error) {
	cacheId := fmt.Sprintf("role:id:%s", convert.UUIDToString(id))
	var role models.RoleEntity
	err := r.c.Get(ctx, cacheId, &role)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, role, constants.NormalCacheDuration)
		return &role, nil
	}

	row, err := r.q.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	role = models.RoleEntity{
		ID:        convert.UUIDToString(row.ID),
		Name:      row.Name,
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, cacheId, role, constants.NormalCacheDuration)

	return &role, nil
}

func (r *roleRepository) GetByname(ctx context.Context, name string) (*models.RoleEntity, error) {
	cacheId := fmt.Sprintf("role:name:%s", name)
	var role models.RoleEntity
	err := r.c.Get(ctx, cacheId, &role)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, role, constants.NormalCacheDuration)
		return &role, nil
	}
	row, err := r.q.GetRoleByName(ctx, name)
	if err != nil {
		return nil, err
	}
	role = models.RoleEntity{
		ID:        convert.UUIDToString(row.ID),
		Name:      row.Name,
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	_ = r.c.Set(ctx, cacheId, role, constants.NormalCacheDuration)

	return &role, nil
}

func (r *roleRepository) Create(ctx context.Context, name string) (*models.RoleEntity, error) {
	row, err := r.q.CreateRole(ctx, name)
	if err != nil {
		return nil, err
	}
		go func() {
	bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "role:all*")
	}()

	role := models.RoleEntity{
		ID:        convert.UUIDToString(row.ID),
		Name:      row.Name,
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	mapCache := map[string]any{
		fmt.Sprintf("role:name:%s", name):                       role,
		fmt.Sprintf("role:id:%s", convert.UUIDToString(row.ID)): role,
	}
	_ = r.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
	return &role, nil
}

func (r *roleRepository) Update(ctx context.Context, params sqlc.UpdateRoleParams) (*models.RoleEntity, error) {
	row, err := r.q.UpdateRole(ctx, params)
	if err != nil {
		return nil, err
	}
	role := models.RoleEntity{
		ID:        convert.UUIDToString(row.ID),
		Name:      row.Name,
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	mapCache := map[string]any{
		fmt.Sprintf("role:name:%s", row.Name):                   role,
		fmt.Sprintf("role:id:%s", convert.UUIDToString(row.ID)): role,
	}
	_ = r.c.MSet(ctx, mapCache, constants.NormalCacheDuration)
	return &role, nil
}

func (r *roleRepository) All(ctx context.Context) ([]*models.RoleEntity, error) {
	queryKey := "role:all"
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		listItem, err := r.getByIDsWithFallback(ctx, cachedIDs)
		if err != nil {
			return nil, err
		}
		newCachedIDs := make([]string, len(listItem))
		for i, media := range listItem {
			newCachedIDs[i] = media.ID
		}
		_ = r.c.Set(ctx, queryKey, newCachedIDs, constants.ListCacheDuration)
		return listItem, err
	}

	rows, err := r.q.GetRoles(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*models.RoleEntity
	var ids []string
	roleToCache := make(map[string]any)

	for _, row := range rows {
		role := &models.RoleEntity{
			ID:        convert.UUIDToString(row.ID),
			Name:      row.Name,
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, role.ID)
		roles = append(roles, role)

		roleToCache[fmt.Sprintf("role:id:%s", role.ID)] = role
	}

	if len(roleToCache) > 0 {
		_ = r.c.MSet(ctx, roleToCache, constants.NormalCacheDuration)
	}

	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return roles, nil
}

func (r *roleRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	role, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = r.q.DeleteRole(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("role:id:%s", role.ID), fmt.Sprintf("role:name:%s", role.Name))
	return nil
}

func (r *roleRepository) Restore(ctx context.Context, id pgtype.UUID) error {
	err := r.q.RestoreRole(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("role:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *roleRepository) AddUserRole(ctx context.Context, params sqlc.AddUserRoleParams) error {
	err := r.q.AddUserRole(ctx, params)
	return err
}

func (r *roleRepository) RemoveUserRole(ctx context.Context, params sqlc.RemoveUserRoleParams) error {
	err := r.q.RemoveUserRole(ctx, params)
	return err
}

func (r *roleRepository) RemoveAllUsersFromRole(ctx context.Context, roleId pgtype.UUID) error {
	err := r.q.RemoveAllUsersFromRole(ctx, roleId)
	return err
}

func (r *roleRepository) RemoveAllRolesFromUser(ctx context.Context, roleId pgtype.UUID) error {
	err := r.q.RemoveAllRolesFromUser(ctx, roleId)
	return err
}

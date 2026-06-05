package repositories

import (
	"context"
	json "history-api/pkg/jsonx"

	"github.com/jackc/pgx/v5"
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
	GetByName(ctx context.Context, name string) (*models.RoleEntity, error)
	All(ctx context.Context) ([]*models.RoleEntity, error)
	Create(ctx context.Context, name string) (*models.RoleEntity, error)
	Update(ctx context.Context, params sqlc.UpdateRoleParams) (*models.RoleEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	Restore(ctx context.Context, id pgtype.UUID) error
	CreateUserRole(ctx context.Context, params sqlc.CreateUserRoleParams) error
	DeleteUserRole(ctx context.Context, params sqlc.DeleteUserRoleParams) error
	BulkDeleteRolesFromUser(ctx context.Context, userId pgtype.UUID) error
	BulkDeleteUsersFromRole(ctx context.Context, roleId pgtype.UUID) error
	WithTx(tx pgx.Tx) RoleRepository
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

func (r *roleRepository) WithTx(tx pgx.Tx) RoleRepository {
	return &roleRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *roleRepository) generateQueryKey(prefix string, params any) string {
	return cache.QueryKey(prefix, params)
}

func (r *roleRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.RoleEntity, error) {
	if len(ids) == 0 {
		return []*models.RoleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("role:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	roles := make([]*models.RoleEntity, 0, len(ids))
	missingRolesToCache := make(map[string]any, len(ids))

	missingPgIds := make([]pgtype.UUID, 0, len(ids))
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.RoleEntity, len(missingPgIds))
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetRolesByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.RoleEntity{
					ID:        convert.UUIDToString(row.ID),
					Name:      row.Name,
					IsDeleted: row.IsDeleted,
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
					UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.RoleEntity
			if err := json.Unmarshal(b, &u); err == nil {
				roles = append(roles, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				roles = append(roles, item)
				missingRolesToCache[keys[i]] = item
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
	cacheId := cache.Key("role:id", convert.UUIDToString(id))
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

func (r *roleRepository) GetByName(ctx context.Context, name string) (*models.RoleEntity, error) {
	cacheId := cache.Key("role:name", name)
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

	_ = r.c.Del(ctx, cache.Key("role:id", convert.UUIDToString(row.ID)), cache.Key("role:name", row.Name))
	return &role, nil
}

func (r *roleRepository) All(ctx context.Context) ([]*models.RoleEntity, error) {
	queryKey := "role:all"
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.RoleEntity{}, nil
		}
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
	roles := make([]*models.RoleEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	roleToCache := make(map[string]any, len(rows))

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

		roleToCache[cache.Key("role:id", role.ID)] = role
	}

	if len(roleToCache) > 0 {
		_ = r.c.MSet(ctx, roleToCache, constants.NormalCacheDuration)
	}

	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

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
	_ = r.c.Del(ctx, cache.Key("role:id", role.ID), cache.Key("role:name", role.Name))
	return nil
}

func (r *roleRepository) Restore(ctx context.Context, id pgtype.UUID) error {
	err := r.q.RestoreRole(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, cache.Key("role:id", convert.UUIDToString(id)))
	return nil
}

func (r *roleRepository) CreateUserRole(ctx context.Context, params sqlc.CreateUserRoleParams) error {
	err := r.q.CreateUserRole(ctx, params)
	return err
}

func (r *roleRepository) DeleteUserRole(ctx context.Context, params sqlc.DeleteUserRoleParams) error {
	err := r.q.DeleteUserRole(ctx, params)
	return err
}

func (r *roleRepository) BulkDeleteUsersFromRole(ctx context.Context, roleId pgtype.UUID) error {
	err := r.q.BulkDeleteUsersFromRole(ctx, roleId)
	return err
}

func (r *roleRepository) BulkDeleteRolesFromUser(ctx context.Context, roleId pgtype.UUID) error {
	err := r.q.BulkDeleteRolesFromUser(ctx, roleId)
	return err
}

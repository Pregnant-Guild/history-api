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

type RoleRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.RoleEntity, error)
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

func (r *roleRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.RoleEntity, error) {
	cacheId := fmt.Sprintf("role:id:%s", convert.UUIDToString(id))
	var role models.RoleEntity
	err := r.c.Get(ctx, cacheId, &role)
	if err == nil {
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
	_ = r.c.Set(ctx, cacheId, role, 5*time.Minute)

	return &role, nil
}

func (r *roleRepository) GetByname(ctx context.Context, name string) (*models.RoleEntity, error) {
	cacheId := fmt.Sprintf("role:name:%s", name)
	var role models.RoleEntity
	err := r.c.Get(ctx, cacheId, &role)
	if err == nil {
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

	_ = r.c.Set(ctx, cacheId, role, 5*time.Minute)

	return &role, nil
}

func (r *roleRepository) Create(ctx context.Context, name string) (*models.RoleEntity, error) {
	row, err := r.q.CreateRole(ctx, name)
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
		fmt.Sprintf("role:name:%s", name):                       role,
		fmt.Sprintf("role:id:%s", convert.UUIDToString(row.ID)): role,
	}
	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
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
	_ = r.c.MSet(ctx, mapCache, 5*time.Minute)
	return &role, nil
}

func (r *roleRepository) All(ctx context.Context) ([]*models.RoleEntity, error) {
	rows, err := r.q.GetRoles(ctx)
	if err != nil {
		return nil, err
	}

	var users []*models.RoleEntity
	for _, row := range rows {
		user := &models.RoleEntity{
			ID:        convert.UUIDToString(row.ID),
			Name:      row.Name,
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		users = append(users, user)
	}

	return users, nil
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

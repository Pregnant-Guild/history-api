package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"history-api/internal/gen/sqlc"
	"history-api/pkg/models"
)

type UserRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	All(ctx context.Context) ([]*models.UserEntity, error)
	Create(ctx context.Context, params sqlc.CreateUserParams) (*models.UserEntity, error)
	Update(ctx context.Context, params sqlc.UpdateUserParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context,  params sqlc.UpdateUserPasswordParams) error
	ExistEmail(ctx context.Context, email string) (bool, error)
	Verify(ctx context.Context,  id pgtype.UUID) error
	Delete(ctx context.Context,  id pgtype.UUID) error
	Restore(ctx context.Context,  id pgtype.UUID) error
}

type userRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(db sqlc.DBTX) UserRepository {
	return &userRepository{
		q: sqlc.New(db),
	}
}

func (r *userRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.UserEntity, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user := &models.UserEntity{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		AvatarUrl:    row.AvatarUrl,
		IsActive:     row.IsActive,
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if err := user.ParseRoles(row.Roles); err != nil {
			return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	user := &models.UserEntity{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		AvatarUrl:    row.AvatarUrl,
		IsActive:     row.IsActive,
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if err := user.ParseRoles(row.Roles); err != nil {
			return nil, err
	}
	return user, nil
}

func (r *userRepository) Create(ctx context.Context, params sqlc.CreateUserParams) (*models.UserEntity, error) {
	row, err := r.q.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return &models.UserEntity{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		AvatarUrl:    row.AvatarUrl,
		IsActive:     row.IsActive,
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		RefreshToken: row.RefreshToken,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Roles:        make([]*models.RoleSimple, 0),
	}, nil
}

func (r *userRepository) Update(ctx context.Context, params sqlc.UpdateUserParams) (*models.UserEntity, error) {
	row, err := r.q.UpdateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	user := &models.UserEntity{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		AvatarUrl:    row.AvatarUrl,
		IsActive:     row.IsActive,
		IsVerified:   row.IsVerified,
		TokenVersion: row.TokenVersion,
		IsDeleted:    row.IsDeleted,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if err := user.ParseRoles(row.Roles); err != nil {
			return nil, err
	}

	return user, nil
}

func (r *userRepository) All(ctx context.Context) ([]*models.UserEntity, error) {
	rows, err := r.q.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	var users []*models.UserEntity
	for _, row := range rows {
		user := &models.UserEntity{
			ID:           row.ID,
			Name:         row.Name,
			Email:        row.Email,
			PasswordHash: row.PasswordHash,
			AvatarUrl:    row.AvatarUrl,
			IsActive:     row.IsActive,
			IsVerified:   row.IsVerified,
			TokenVersion: row.TokenVersion,
			IsDeleted:    row.IsDeleted,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}

		if err := user.ParseRoles(row.Roles); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) Verify(ctx context.Context, id pgtype.UUID) error {
	err := r.q.VerifyUser(ctx, id)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteUser(ctx, id)
	return err
}

func (r *userRepository) Restore(ctx context.Context, id pgtype.UUID) error {
	err := r.q.RestoreUser(ctx, id)
	return err
}

func (r *userRepository) UpdatePassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error {
	err := r.q.UpdateUserPassword(ctx, params)
	return err
}

func (r *userRepository) ExistEmail(ctx context.Context, email string) (bool, error) {
	row, err := r.q.ExistsUserByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	return  row, nil
}
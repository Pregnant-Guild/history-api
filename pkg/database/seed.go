package database

import (
	"context"
	"database/sql"
	"errors"
	"history-api/internal/gen/sqlc"
	"history-api/pkg/config"
	"history-api/pkg/constants"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func SeedSuperAdmin(pool *pgxpool.Pool) error {
	ctx := context.Background()

	displayName, err := config.GetConfig("ADMIN_DISPLAY_NAME")
	if err != nil {
		return err
	}

	email, err := config.GetConfig("ADMIN_EMAIL")
	if err != nil {
		return err
	}

	password, err := config.GetConfig("ADMIN_PASSWORD")
	if err != nil {
		return err
	}

	q := sqlc.New(pool)
	

	_, err = q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	repo := q.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email: email,
		PasswordHash: pgtype.Text{
			String: string(hashed),
			Valid:  len(hashed) != 0,
		},
		AuthProvider: constants.ProviderTypeLocal.String(),
	})
	if err != nil {
		return err
	}

	_, err = repo.CreateUserProfile(ctx, sqlc.CreateUserProfileParams{
		UserID: user.ID,
		DisplayName: pgtype.Text{
			String: displayName,
			Valid:  displayName != "",
		},
	})
	if err != nil {
		return err
	}

	adminRole, err := q.GetRoleByName(ctx, constants.RoleTypeAdmin.String())
	if err != nil {
		return err
	}

	useRole, err := q.GetRoleByName(ctx, constants.RoleTypeUser.String())
	if err != nil {
		return err
	}

	err = repo.CreateUserRole(
		ctx,
		sqlc.CreateUserRoleParams{
			UserID:  user.ID,
			Column2: []pgtype.UUID{adminRole.ID, useRole.ID},
		},
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

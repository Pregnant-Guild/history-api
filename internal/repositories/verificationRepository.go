package repositories

import (
	"context"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"

	"github.com/jackc/pgx/v5/pgtype"
)

type VerificationRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.UserVerificationEntity, error)
	GetByUserID(ctx context.Context, id pgtype.UUID) ([]*models.UserVerificationEntity, error)
	Count(ctx context.Context, params sqlc.CountUserVerificationsParams) (int64, error)
	Search(ctx context.Context, params sqlc.SearchUserVerificationsParams) ([]*models.UserVerificationEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateVerificationMedia(ctx context.Context, params sqlc.CreateVerificationMediaParams) error
	DeleteVerificationMedia(ctx context.Context, params sqlc.DeleteVerificationMediasParams) error
}

type verificationRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

// func NewVerificationRepository(db sqlc.DBTX, c cache.Cache) VerificationRepository {
// 	return &verificationRepository{
// 		q: sqlc.New(db),
// 		c: c,
// 	}
// }

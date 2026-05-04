package repositories

import (
	"context"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/convert"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

type RagRepository interface {
	SaveChunk(ctx context.Context, sourceType string, sourceID string, projectID string, index int, content string, vector []float32) error
	SearchSimilar(ctx context.Context, projectID *string, vector []float32, limit int, threshold float64) ([]*models.RagChunk, error)
	DeleteBySourceIDs(ctx context.Context, sourceType string, sourceIDs []string) error
	WithTx(tx pgx.Tx) RagRepository
}

type ragRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewRagRepository(db sqlc.DBTX, c cache.Cache) RagRepository {
	return &ragRepository{q: sqlc.New(db), c: c}
}

func (r *ragRepository) WithTx(tx pgx.Tx) RagRepository {
	return &ragRepository{q: r.q.WithTx(tx), c: r.c}
}

func (r *ragRepository) SaveChunk(ctx context.Context, sourceType string, sourceID string, projectID string, index int, content string, vector []float32) error {
	pID, _ := convert.StringToUUID(projectID)
	sID, _ := convert.StringToUUID(sourceID)

	_, err := r.q.CreateRagChunk(ctx, sqlc.CreateRagChunkParams{
		SourceType: sourceType,
		SourceID:   sID,
		ProjectID:  pID,
		ChunkIndex: int32(index),
		Content:    content,
		Embedding:  pgvector.NewVector(vector),
	})
	return err
}

func (r *ragRepository) SearchSimilar(ctx context.Context, projectID *string, vector []float32, limit int, threshold float64) ([]*models.RagChunk, error) {
	params := sqlc.SearchRagChunksParams{
		Embedding:      pgvector.NewVector(vector),
		MatchThreshold: threshold,
		MatchCount:     int32(limit),
	}
	if projectID != nil && *projectID != "" {
		pID, _ := convert.StringToUUID(*projectID)
		params.ProjectID = pID
	}

	rows, err := r.q.SearchRagChunks(ctx, params)
	if err != nil {
		return nil, err
	}

	res := make([]*models.RagChunk, len(rows))
	for i, row := range rows {
		res[i] = &models.RagChunk{
			ID:         convert.UUIDToString(row.ID),
			Content:    row.Content,
			Similarity: row.Similarity,
		}
	}
	return res, nil
}

func (r *ragRepository) DeleteBySourceIDs(ctx context.Context, sourceType string, sourceIDs []string) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	
	uids := make([]pgtype.UUID, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		uid, err := convert.StringToUUID(id)
		if err == nil {
			uids = append(uids, uid)
		}
	}
	
	return r.q.DeleteRagChunksBySourceIDs(ctx, sqlc.DeleteRagChunksBySourceIDsParams{
		SourceType: sourceType,
		Column2:  uids,
	})
}

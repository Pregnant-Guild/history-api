package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	CreateConversation(ctx context.Context, params sqlc.CreateConversationParams) (*models.ConversationEntity, error)
	UpdateConversationStatus(ctx context.Context, params sqlc.UpdateConversationStatusParams) (*models.ConversationEntity, error)
	CreateMessage(ctx context.Context, params sqlc.CreateMessageParams) (*models.MessageEntity, error)
	GetMessagesByConversation(ctx context.Context, params sqlc.GetMessagesByConversationParams) ([]*models.MessageEntity, error)
	CreateChatbotHistory(ctx context.Context, params sqlc.CreateChatbotHistoryParams) (*models.ChatbotHistoryEntity, error)
	GetChatbotHistory(ctx context.Context, params sqlc.GetChatbotHistoryParams) ([]*models.ChatbotHistoryEntity, error)
}

type chatRepository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
	c  cache.Cache
}

func NewChatRepository(db *pgxpool.Pool, c cache.Cache) ChatRepository {
	return &chatRepository{
		db: db,
		q:  sqlc.New(db),
		c:  c,
	}
}

func (r *chatRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:query:%s", prefix, hash)
}

func (r *chatRepository) getConversationsByIDsWithFallback(ctx context.Context, ids []string) ([]*models.ConversationEntity, error) {
	if len(ids) == 0 {
		return []*models.ConversationEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("conversation:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var results []*models.ConversationEntity
	missingToCache := make(map[string]any)

	var missingPgIds []pgtype.UUID
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.ConversationEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetConversationsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.ConversationEntity{
					ID:        convert.UUIDToString(row.ID),
					UserID:    convert.UUIDToString(row.UserID),
					ModID:     convert.UUIDToStringPtr(row.ModID),
					Status:    row.Status,
					ClosedAt:  convert.TimeToPtr(row.ClosedAt),
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
					UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var c models.ConversationEntity
			if err := json.Unmarshal(b, &c); err == nil {
				results = append(results, &c)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				results = append(results, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return results, nil
}

func (r *chatRepository) getMessagesByIDsWithFallback(ctx context.Context, ids []string) ([]*models.MessageEntity, error) {
	if len(ids) == 0 {
		return []*models.MessageEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("message:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var results []*models.MessageEntity
	missingToCache := make(map[string]any)

	var missingPgIds []pgtype.UUID
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.MessageEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetMessagesByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.MessageEntity{
					ID:             convert.UUIDToString(row.ID),
					ConversationID: convert.UUIDToString(row.ConversationID),
					SenderID:       convert.UUIDToString(row.SenderID),
					Content:        row.Content,
					CreatedAt:      convert.TimeToPtr(row.CreatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var c models.MessageEntity
			if err := json.Unmarshal(b, &c); err == nil {
				results = append(results, &c)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				results = append(results, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return results, nil
}

func (r *chatRepository) getChatbotHistoriesByIDsWithFallback(ctx context.Context, ids []string) ([]*models.ChatbotHistoryEntity, error) {
	if len(ids) == 0 {
		return []*models.ChatbotHistoryEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("chatbot_history:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var results []*models.ChatbotHistoryEntity
	missingToCache := make(map[string]any)

	var missingPgIds []pgtype.UUID
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.ChatbotHistoryEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetChatbotHistoriesByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.ChatbotHistoryEntity{
					ID:        convert.UUIDToString(row.ID),
					UserID:    convert.UUIDToString(row.UserID),
					Question:  row.Question,
					Answer:    row.Answer,
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var c models.ChatbotHistoryEntity
			if err := json.Unmarshal(b, &c); err == nil {
				results = append(results, &c)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				results = append(results, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return results, nil
}

func (r *chatRepository) CreateConversation(ctx context.Context, params sqlc.CreateConversationParams) (*models.ConversationEntity, error) {
	row, err := r.q.CreateConversation(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := &models.ConversationEntity{
		ID:        convert.UUIDToString(row.ID),
		UserID:    convert.UUIDToString(row.UserID),
		ModID:     convert.UUIDToStringPtr(row.ModID),
		Status:    row.Status,
		ClosedAt:  convert.TimeToPtr(row.ClosedAt),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	return entity, nil
}

func (r *chatRepository) UpdateConversationStatus(ctx context.Context, params sqlc.UpdateConversationStatusParams) (*models.ConversationEntity, error) {
	row, err := r.q.UpdateConversationStatus(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := &models.ConversationEntity{
		ID:        convert.UUIDToString(row.ID),
		UserID:    convert.UUIDToString(row.UserID),
		ModID:     convert.UUIDToStringPtr(row.ModID),
		Status:    row.Status,
		ClosedAt:  convert.TimeToPtr(row.ClosedAt),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Del(ctx, fmt.Sprintf("conversation:id:%s", entity.ID))
	return entity, nil
}

func (r *chatRepository) CreateMessage(ctx context.Context, params sqlc.CreateMessageParams) (*models.MessageEntity, error) {
	row, err := r.q.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := &models.MessageEntity{
		ID:             convert.UUIDToString(row.ID),
		ConversationID: convert.UUIDToString(row.ConversationID),
		SenderID:       convert.UUIDToString(row.SenderID),
		Content:        row.Content,
		CreatedAt:      convert.TimeToPtr(row.CreatedAt),
	}

	return entity, nil
}

func (r *chatRepository) GetMessagesByConversation(ctx context.Context, params sqlc.GetMessagesByConversationParams) ([]*models.MessageEntity, error) {
	queryKey := r.generateQueryKey("message:conversation", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.MessageEntity{}, nil
		}
		return r.getMessagesByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetMessagesByConversation(ctx, params)
	if err != nil {
		return nil, err
	}

	var results []*models.MessageEntity
	var ids []string
	toCache := make(map[string]any)

	for _, row := range rows {
		item := &models.MessageEntity{
			ID:             convert.UUIDToString(row.ID),
			ConversationID: convert.UUIDToString(row.ConversationID),
			SenderID:       convert.UUIDToString(row.SenderID),
			Content:        row.Content,
			CreatedAt:      convert.TimeToPtr(row.CreatedAt),
		}
		ids = append(ids, item.ID)
		results = append(results, item)
		toCache[fmt.Sprintf("message:id:%s", item.ID)] = item
	}

	if len(toCache) > 0 {
		_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return results, nil
}

func (r *chatRepository) CreateChatbotHistory(ctx context.Context, params sqlc.CreateChatbotHistoryParams) (*models.ChatbotHistoryEntity, error) {
	row, err := r.q.CreateChatbotHistory(ctx, params)
	if err != nil {
		return nil, err
	}

	entity := &models.ChatbotHistoryEntity{
		ID:        convert.UUIDToString(row.ID),
		UserID:    convert.UUIDToString(row.UserID),
		Question:  row.Question,
		Answer:    row.Answer,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
	}

	go func() {
		userId := convert.UUIDToString(params.UserID)
		if userId != "" {
			_ = r.c.DelByPattern(context.Background(), fmt.Sprintf("chatbot_history:userId:%s:*", userId))
		}
	}()

	return entity, nil
}

func (r *chatRepository) GetChatbotHistory(ctx context.Context, params sqlc.GetChatbotHistoryParams) ([]*models.ChatbotHistoryEntity, error) {
	queryKey := fmt.Sprintf(
		"chatbot_history:userId:%s:limit:%d:cursor:%s",
		convert.UUIDToString(params.UserID),
		params.Limit,
		convert.UUIDToString(params.CursorID),
	)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.ChatbotHistoryEntity{}, nil
		}
		return r.getChatbotHistoriesByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetChatbotHistory(ctx, params)
	if err != nil {
		return nil, err
	}

	var results []*models.ChatbotHistoryEntity
	var ids []string
	toCache := make(map[string]any)

	for _, row := range rows {
		item := &models.ChatbotHistoryEntity{
			ID:        convert.UUIDToString(row.ID),
			UserID:    convert.UUIDToString(row.UserID),
			Question:  row.Question,
			Answer:    row.Answer,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
		}
		ids = append(ids, item.ID)
		results = append(results, item)
		toCache[fmt.Sprintf("chatbot_history:id:%s", item.ID)] = item
	}

	if len(toCache) > 0 {
		_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return results, nil
}

package repositories

import (
	"context"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	json "history-api/pkg/jsonx"
	"strconv"

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
	return cache.QueryKey(prefix, params)
}

func (r *chatRepository) getConversationsByIDsWithFallback(ctx context.Context, ids []string) ([]*models.ConversationEntity, error) {
	if len(ids) == 0 {
		return []*models.ConversationEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("conversation:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	results := make([]*models.ConversationEntity, 0, len(ids))
	missingToCache := make(map[string]any, len(ids))

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

	dbMap := make(map[string]*models.ConversationEntity, len(missingPgIds))
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
		keys[i] = cache.Key("message:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	results := make([]*models.MessageEntity, 0, len(ids))
	missingToCache := make(map[string]any, len(ids))

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

	dbMap := make(map[string]*models.MessageEntity, len(missingPgIds))
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
		keys[i] = cache.Key("chatbot_history:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	results := make([]*models.ChatbotHistoryEntity, 0, len(ids))
	missingToCache := make(map[string]any, len(ids))

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

	dbMap := make(map[string]*models.ChatbotHistoryEntity, len(missingPgIds))
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
	_ = r.c.Del(ctx, cache.Key("conversation:id", entity.ID))
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

	results := make([]*models.MessageEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	toCache := make(map[string]any, len(rows))

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
		toCache[cache.Key("message:id", item.ID)] = item
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
			_ = r.c.DelByPattern(context.Background(), "chatbot_history:userId:"+userId+":*")
		}
	}()

	return entity, nil
}

func (r *chatRepository) GetChatbotHistory(ctx context.Context, params sqlc.GetChatbotHistoryParams) ([]*models.ChatbotHistoryEntity, error) {
	queryKey := "chatbot_history:userId:" + convert.UUIDToString(params.UserID) +
		":limit:" + strconv.Itoa(int(params.Limit)) +
		":cursor:" + convert.UUIDToString(params.CursorID)
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

	results := make([]*models.ChatbotHistoryEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	toCache := make(map[string]any, len(rows))

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
		toCache[cache.Key("chatbot_history:id", item.ID)] = item
	}

	if len(toCache) > 0 {
		_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return results, nil
}

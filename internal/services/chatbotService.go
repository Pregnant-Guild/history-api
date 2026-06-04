package services

import (
	"context"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/ai"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type ChatbotService interface {
	Chat(ctx context.Context, userID string, projectID *string, question string) (string, error)
	GetHistory(ctx context.Context, userID string, dto *request.GetChatbotHistoryDto) ([]*models.ChatbotHistoryEntity, error)
}

type chatbotService struct {
	repo      repositories.RagRepository
	usageRepo repositories.UsageRepository
	chatRepo  repositories.ChatRepository
	ragUtils  *ai.RagUtils
}

func NewChatbotService(repo repositories.RagRepository, usageRepo repositories.UsageRepository, chatRepo repositories.ChatRepository, ragUtils *ai.RagUtils) ChatbotService {
	return &chatbotService{
		repo:      repo,
		usageRepo: usageRepo,
		chatRepo:  chatRepo,
		ragUtils:  ragUtils,
	}
}

func (s *chatbotService) Chat(ctx context.Context, userID string, projectID *string, question string) (string, error) {
	usage, err := s.usageRepo.GetAIUsage(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to check usage: %w", err)
	}

	if usage >= constants.MaxDailyAIUsage {
		return "", fmt.Errorf("you have reached your daily limit of %d questions. Please come back tomorrow", constants.MaxDailyAIUsage)
	}

	qVector, err := s.ragUtils.EmbedQuery(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to embed question: %w", err)
	}

	results, err := s.repo.SearchSimilar(ctx, projectID, qVector, 8, 0.50)
	if err != nil {
		return "", fmt.Errorf("failed to search similar content: %w", err)
	}

	if len(results) < 3 {
		broadResults, err := s.repo.SearchSimilar(ctx, projectID, qVector, 8, 0.35)
		if err == nil && len(broadResults) > len(results) {
			results = broadResults
		}
	}

	var contextBuilder strings.Builder
	for i, res := range results {
		contextBuilder.WriteString(fmt.Sprintf("[Document %d (score: %.2f)]: %s\n", i+1, res.Similarity, res.Content))
	}
	contextStr := contextBuilder.String()

	pgUserID, err := convert.StringToUUID(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	histories, err := s.chatRepo.GetChatbotHistory(ctx, sqlc.GetChatbotHistoryParams{
		UserID: pgUserID,
		Limit:  10,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to get chatbot history")
	}

	var historyBuilder strings.Builder
	for _, h := range histories {
		historyBuilder.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n\n", h.Question, h.Answer))
	}
	historyStr := historyBuilder.String()

	var prompt string
	if contextStr == "" {
		prompt = fmt.Sprintf(`You are a friendly history assistant chatbot.

Recent Chat History:
%s

The user said: "%s"

Rules:
- You MUST reply in the same language as the user's question (e.g., if the user greets or asks in Vietnamese, reply in Vietnamese).
- If it is a greeting (like "hello", "hi", "xin chào"), respond with a friendly greeting and briefly introduce yourself.
- If it is a history question, say that you don't have relevant documents to answer.
- You MUST wrap your final response inside <answer> tags. Example: <answer>Hello!</answer>
- Do NOT show your reasoning outside or inside the tags if possible, but the final answer MUST be in <answer> tags.`, historyStr, question)
	} else {
		prompt = fmt.Sprintf(`You are a helpful history assistant. Answer the question using ONLY the provided context.

Rules:
- You MUST reply in the same language as the user's question (e.g., if the question is in Vietnamese, reply in Vietnamese).
- If the answer is not in the context, say "I don't have enough historical context to answer that."
- You MUST wrap your final response inside <answer> tags. Example: <answer>The capital is...</answer>
- Be concise and direct.

Context:
%s

Recent Chat History:
%s

Question: %s`, contextStr, historyStr, question)
	}

	response, err := s.ragUtils.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}
	if _, err := s.usageRepo.IncrementAIUsage(ctx, userID); err != nil {
		log.Warn().Err(err).Str("userID", userID).Msg("failed to increment AI usage")
	}

	_, err = s.chatRepo.CreateChatbotHistory(ctx, sqlc.CreateChatbotHistoryParams{
		UserID:   pgUserID,
		Question: question,
		Answer:   response,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to save chatbot history")
	}

	return response, nil
}

func (s *chatbotService) GetHistory(ctx context.Context, userID string, dto *request.GetChatbotHistoryDto) ([]*models.ChatbotHistoryEntity, error) {
	pgUserID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	var pgCursorID pgtype.UUID
	if dto.Cursor != nil {
		if err := pgCursorID.Scan(*dto.Cursor); err != nil {
			return nil, fmt.Errorf("invalid cursor id: %w", err)
		}
	} else {
		pgCursorID.Valid = false
	}

	if dto.Limit <= 0 {
		dto.Limit = 10
	}

	return s.chatRepo.GetChatbotHistory(ctx, sqlc.GetChatbotHistoryParams{
		UserID:   pgUserID,
		CursorID: pgCursorID,
		Limit:    int32(dto.Limit),
	})
}

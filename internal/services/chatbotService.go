package services

import (
	"context"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/ai"
	"history-api/pkg/config"
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

	pgUserID, err := convert.StringToUUID(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	searchQuery := strings.TrimSpace(question)
	rewriteHistory := s.getRewriteHistory(ctx, pgUserID)
	rewrittenQuery, err := s.ragUtils.RewriteSearchQuery(ctx, question, rewriteHistory)
	if err != nil {
		log.Warn().Err(err).Msg("failed to rewrite RAG search query")
	} else if strings.TrimSpace(rewrittenQuery) != "" {
		searchQuery = strings.TrimSpace(rewrittenQuery)
	}

	candidateLimit := config.GetIntConfigWithDefault("RAG_RETRIEVAL_CANDIDATES", 30)
	if candidateLimit < 8 {
		candidateLimit = 8
	}

	contextLimit := config.GetIntConfigWithDefault("RAG_CONTEXT_TOP_N", 8)
	if contextLimit <= 0 {
		contextLimit = 8
	}
	if contextLimit > candidateLimit {
		contextLimit = candidateLimit
	}

	qVector, err := s.ragUtils.EmbedQuery(ctx, searchQuery)
	if err != nil {
		return "", fmt.Errorf("failed to embed question: %w", err)
	}

	results, err := s.repo.SearchSimilar(ctx, projectID, qVector, candidateLimit, 0.45)
	if err != nil {
		return "", fmt.Errorf("failed to search similar content: %w", err)
	}

	if len(results) < contextLimit {
		broadResults, err := s.repo.SearchSimilar(ctx, projectID, qVector, candidateLimit, 0.30)
		if err == nil && len(broadResults) > len(results) {
			results = broadResults
		}
	}

	results = s.rerankResults(ctx, searchQuery, results, contextLimit)

	var contextBuilder strings.Builder
	contextBuilder.Grow(len(results) * 128)
	for i, res := range results {
		contextBuilder.WriteString(fmt.Sprintf("<doc id=\"%d\" score=\"%.2f\">\n%s\n</doc>\n\n", i+1, res.Similarity, res.Content))
	}
	contextStr := strings.TrimSpace(contextBuilder.String())

	var prompt string
	if contextStr == "" {
		prompt = fmt.Sprintf(`You are a friendly history assistant chatbot.

User Question:
%s

Rules:
- Reply in the same language as the user's question.
- If the user is greeting, respond with a friendly greeting and briefly introduce yourself.
- If the user asks a history-related question, respond in the same language as the user's question, stating that you don't have enough historical context to answer that (for example, in Vietnamese: "Tôi không có đủ dữ liệu lịch sử để trả lời câu hỏi này.").
- Do not answer historical questions from memory.
- Do not use your own knowledge, assumptions, memory, or external facts.
- Do not guess, infer, assume, or invent missing information.
- Your final response MUST be wrapped inside <answer> tags.
- Do not output anything outside <answer> tags.`, question)
	} else {
		prompt = fmt.Sprintf(`You are a retrieval-augmented history assistant.

Context:
%s

Question:
%s

Rules:
- Reply in the same language as the user's question.
- Use ONLY the information explicitly stated in Context.
- Treat Context as the only source of truth.
- Never use your own knowledge, assumptions, memory, chat history, or external facts.
- Never infer information that is not explicitly stated in Context.
- Never create names, dates, places, events, causes, results, or explanations that are not in Context.
- Every factual sentence must be directly supported by Context.
- If Context does not contain enough information to answer, respond in the same language as the user's question, stating that you don't have enough historical context to answer that (for example, in Vietnamese: "Tôi không có đủ dữ liệu lịch sử để trả lời câu hỏi này.").
- If Context only partially answers the question, answer only the supported part and clearly say the remaining information is not available in the provided context.
- Do not mention document scores.
- Do not cite documents unless the user asks.
- Your final response MUST be wrapped inside <answer> tags.
- Do not output anything outside <answer> tags.
- Before writing the final answer, silently verify that each factual sentence is supported by Context.
- Keep the answer focused on the user's question. Do not add background information that is not needed.
- Answer in complete, natural, grammatically correct sentences.`, contextStr, question)
	}

	response, err := s.ragUtils.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	response = normalizeAnswer(response)

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

func (s *chatbotService) getRewriteHistory(ctx context.Context, userID pgtype.UUID) []ai.ChatTurn {
	limit := config.GetIntConfigWithDefault("RAG_REWRITE_HISTORY_TURNS", 3)
	if limit <= 0 {
		return nil
	}
	if limit > 5 {
		limit = 5
	}

	history, err := s.chatRepo.GetChatbotHistory(ctx, sqlc.GetChatbotHistoryParams{
		UserID: userID,
		Limit:  int32(limit),
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to load chatbot history for RAG query rewrite")
		return nil
	}

	turns := make([]ai.ChatTurn, 0, len(history))
	for _, item := range history {
		if item == nil {
			continue
		}
		turns = append(turns, ai.ChatTurn{
			Question: item.Question,
			Answer:   item.Answer,
		})
	}

	return turns
}

func (s *chatbotService) rerankResults(ctx context.Context, query string, results []*models.RagChunk, limit int) []*models.RagChunk {
	if len(results) == 0 {
		return results
	}
	if limit <= 0 {
		limit = len(results)
	}
	if limit > len(results) {
		limit = len(results)
	}

	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	ranked, err := s.ragUtils.RerankDocuments(ctx, query, documents, limit)
	if err != nil {
		log.Warn().Err(err).Msg("failed to rerank RAG results")
		return limitRagResults(results, limit)
	}
	if len(ranked) == 0 {
		return limitRagResults(results, limit)
	}

	selected := make([]*models.RagChunk, 0, limit)
	seen := make(map[int]struct{}, limit)
	for _, item := range ranked {
		if item.Index < 0 || item.Index >= len(results) {
			continue
		}
		if _, exists := seen[item.Index]; exists {
			continue
		}
		seen[item.Index] = struct{}{}
		selected = append(selected, results[item.Index])
		if len(selected) >= limit {
			return selected
		}
	}

	for i, result := range results {
		if _, exists := seen[i]; exists {
			continue
		}
		selected = append(selected, result)
		if len(selected) >= limit {
			break
		}
	}

	return selected
}

func limitRagResults(results []*models.RagChunk, limit int) []*models.RagChunk {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

func normalizeAnswer(s string) string {
	s = strings.TrimSpace(s)

	start := strings.Index(s, "<answer>")
	end := strings.LastIndex(s, "</answer>")

	if start >= 0 && end > start {
		return strings.TrimSpace(s[start+len("<answer>") : end])
	}

	s = strings.TrimSpace(strings.TrimPrefix(s, "Answer:"))
	s = strings.TrimSpace(strings.TrimPrefix(s, "<answer>"))
	s = strings.TrimSpace(strings.TrimSuffix(s, "</answer>"))

	return s
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

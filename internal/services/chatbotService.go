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
	"time"

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
	totalStart := time.Now()
	projectIDLog := ""
	if projectID != nil {
		projectIDLog = *projectID
	}
	questionLen := len([]rune(question))

	log.Info().
		Str("userID", userID).
		Str("projectID", projectIDLog).
		Int("question_len", questionLen).
		Msg("rag chat started")

	usageStart := time.Now()
	usage, err := s.usageRepo.GetAIUsage(ctx, userID)
	usageDuration := time.Since(usageStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Dur("usage_check_duration", usageDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat failed while checking usage")
		return "", fmt.Errorf("failed to check usage: %w", err)
	}

	if usage >= constants.MaxDailyAIUsage {
		log.Warn().
			Str("userID", userID).
			Int("usage", usage).
			Dur("usage_check_duration", usageDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat rejected by daily usage limit")
		return "", fmt.Errorf("you have reached your daily limit of %d questions. Please come back tomorrow", constants.MaxDailyAIUsage)
	}

	convertStart := time.Now()
	pgUserID, err := convert.StringToUUID(userID)
	convertDuration := time.Since(convertStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Dur("uuid_convert_duration", convertDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat failed while converting user id")
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	searchQuery := strings.TrimSpace(question)
	historyStart := time.Now()
	rewriteHistory := s.getRewriteHistory(ctx, pgUserID)
	historyDuration := time.Since(historyStart)

	rewriteStart := time.Now()
	rewrittenQuery, err := s.ragUtils.RewriteSearchQuery(ctx, question, rewriteHistory)
	rewriteDuration := time.Since(rewriteStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Int("history_turns", len(rewriteHistory)).
			Dur("rewrite_duration", rewriteDuration).
			Msg("failed to rewrite RAG search query")
	} else if strings.TrimSpace(rewrittenQuery) != "" {
		searchQuery = strings.TrimSpace(rewrittenQuery)
	}
	queryRewritten := searchQuery != strings.TrimSpace(question)

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

	embedStart := time.Now()
	qVector, err := s.ragUtils.EmbedQuery(ctx, searchQuery)
	embedDuration := time.Since(embedStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Bool("query_rewritten", queryRewritten).
			Dur("embed_duration", embedDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat failed while embedding query")
		return "", fmt.Errorf("failed to embed question: %w", err)
	}

	searchStart := time.Now()
	results, err := s.repo.SearchSimilar(ctx, projectID, qVector, candidateLimit, 0.45)
	searchDuration := time.Since(searchStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Str("projectID", projectIDLog).
			Int("candidate_limit", candidateLimit).
			Dur("vector_search_duration", searchDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat failed while searching similar content")
		return "", fmt.Errorf("failed to search similar content: %w", err)
	}
	initialResultCount := len(results)

	var broadSearchDuration time.Duration
	broadResultCount := 0
	if len(results) < contextLimit {
		broadSearchStart := time.Now()
		broadResults, err := s.repo.SearchSimilar(ctx, projectID, qVector, candidateLimit, 0.30)
		broadSearchDuration = time.Since(broadSearchStart)
		if err == nil && len(broadResults) > len(results) {
			results = broadResults
		}
		if err != nil {
			log.Warn().
				Err(err).
				Str("userID", userID).
				Str("projectID", projectIDLog).
				Int("candidate_limit", candidateLimit).
				Dur("broad_search_duration", broadSearchDuration).
				Msg("rag broad vector search failed")
		}
		broadResultCount = len(broadResults)
	}

	rerankStart := time.Now()
	results = s.rerankResults(ctx, searchQuery, results, contextLimit)
	rerankDuration := time.Since(rerankStart)
	results = limitRagResultsByContextChars(results, config.GetIntConfigWithDefault("RAG_CONTEXT_MAX_CHARS", 8000))

	promptBuildStart := time.Now()
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
	promptBuildDuration := time.Since(promptBuildStart)

	generateStart := time.Now()
	response, err := s.ragUtils.GenerateResponse(ctx, prompt)
	generateDuration := time.Since(generateStart)
	if err != nil {
		log.Warn().
			Err(err).
			Str("userID", userID).
			Str("projectID", projectIDLog).
			Int("prompt_chars", len(prompt)).
			Dur("generate_duration", generateDuration).
			Dur("total_duration", time.Since(totalStart)).
			Msg("rag chat failed while generating response")
		return "", err
	}

	response = normalizeAnswer(response)

	usageIncrementStart := time.Now()
	if _, err := s.usageRepo.IncrementAIUsage(ctx, userID); err != nil {
		log.Warn().Err(err).Str("userID", userID).Msg("failed to increment AI usage")
	}
	usageIncrementDuration := time.Since(usageIncrementStart)

	historySaveStart := time.Now()
	_, err = s.chatRepo.CreateChatbotHistory(ctx, sqlc.CreateChatbotHistoryParams{
		UserID:   pgUserID,
		Question: question,
		Answer:   response,
	})
	historySaveDuration := time.Since(historySaveStart)
	if err != nil {
		log.Warn().Err(err).Msg("failed to save chatbot history")
	}

	log.Info().
		Str("userID", userID).
		Str("projectID", projectIDLog).
		Int("question_len", questionLen).
		Int("search_query_len", len([]rune(searchQuery))).
		Bool("query_rewritten", queryRewritten).
		Int("history_turns", len(rewriteHistory)).
		Int("candidate_limit", candidateLimit).
		Int("context_limit", contextLimit).
		Int("initial_results", initialResultCount).
		Int("broad_results", broadResultCount).
		Int("final_results", len(results)).
		Int("context_chars", len(contextStr)).
		Int("prompt_chars", len(prompt)).
		Int("answer_chars", len(response)).
		Dur("usage_check_duration", usageDuration).
		Dur("uuid_convert_duration", convertDuration).
		Dur("history_load_duration", historyDuration).
		Dur("rewrite_duration", rewriteDuration).
		Dur("embed_duration", embedDuration).
		Dur("vector_search_duration", searchDuration).
		Dur("broad_search_duration", broadSearchDuration).
		Dur("rerank_duration", rerankDuration).
		Dur("prompt_build_duration", promptBuildDuration).
		Dur("generate_duration", generateDuration).
		Dur("usage_increment_duration", usageIncrementDuration).
		Dur("history_save_duration", historySaveDuration).
		Dur("total_duration", time.Since(totalStart)).
		Msg("rag chat completed")

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

func limitRagResultsByContextChars(results []*models.RagChunk, maxChars int) []*models.RagChunk {
	if maxChars <= 0 || len(results) == 0 {
		return results
	}

	selected := make([]*models.RagChunk, 0, len(results))
	used := 0
	for _, result := range results {
		if result == nil {
			continue
		}

		nextLen := len(result.Content) + 64
		if len(selected) > 0 && used+nextLen > maxChars {
			break
		}

		selected = append(selected, result)
		used += nextLen
	}

	if len(selected) == 0 {
		return results[:1]
	}

	return selected
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

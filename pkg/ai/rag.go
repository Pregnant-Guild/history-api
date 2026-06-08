package ai

import (
	"bytes"
	"context"
	"fmt"
	"history-api/pkg/config"
	json "history-api/pkg/jsonx"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/textsplitter"
)

type RagUtils struct {
	llm                   llms.Model
	embedder              *embeddings.EmbedderImpl
	httpClient            *http.Client
	openRouterAPIKey      string
	model                 string
	fallbackModel         string
	generationMaxRetries  int
	generationRetryDelay  time.Duration
	rerankEnabled         bool
	rerankModel           string
	rerankFallbackModel   string
	rerankMaxRetries      int
	rerankRetryDelay      time.Duration
	queryRewriteEnabled   bool
	queryRewriteModel     string
	queryRewriteTimeout   time.Duration
	queryRewriteMaxTokens int
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

type RerankResult struct {
	Index int
	Score float64
}

type ChatTurn struct {
	Question string
	Answer   string
}

func NewRagUtils() (*RagUtils, error) {
	openRouterAPIKey := config.GetConfigWithDefault("OPEN_ROUTER_API", "")
	if openRouterAPIKey == "" {
		return nil, fmt.Errorf("OPEN_ROUTER_API is not set")
	}

	model := config.GetConfigWithDefault("OPEN_ROUTER_MODEL", "qwen/qwen3.5-flash-02-23")

	embeddingModel := config.GetConfigWithDefault("OPEN_ROUTER_EMBEDDING_MODEL", "qwen/qwen3-embedding-8b")

	llmTimeoutSeconds := config.GetIntConfigWithDefault("RAG_LLM_TIMEOUT_SECONDS", 20)
	if llmTimeoutSeconds <= 0 {
		llmTimeoutSeconds = 20
	}

	llm, err := openai.New(
		openai.WithToken(openRouterAPIKey),
		openai.WithBaseURL(openRouterBaseURL),
		openai.WithModel(model),
		openai.WithEmbeddingModel(embeddingModel),
		openai.WithHTTPClient(&http.Client{Timeout: time.Duration(llmTimeoutSeconds) * time.Second}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init openrouter ai: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to init embedder: %w", err)
	}

	timeoutSeconds := config.GetIntConfigWithDefault("RAG_RERANK_TIMEOUT_SECONDS", 10)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	rerankMaxRetries := config.GetIntConfigWithDefault("RAG_RERANK_MAX_RETRIES", 2)
	if rerankMaxRetries < 0 {
		rerankMaxRetries = 0
	}
	if rerankMaxRetries > 5 {
		rerankMaxRetries = 5
	}

	rerankRetryDelayMs := config.GetIntConfigWithDefault("RAG_RERANK_RETRY_DELAY_MS", 250)
	if rerankRetryDelayMs <= 0 {
		rerankRetryDelayMs = 250
	}

	generationMaxRetries := config.GetIntConfigWithDefault("RAG_GENERATION_MAX_RETRIES", 2)
	if generationMaxRetries < 0 {
		generationMaxRetries = 0
	}
	if generationMaxRetries > 5 {
		generationMaxRetries = 5
	}

	generationRetryDelayMs := config.GetIntConfigWithDefault("RAG_GENERATION_RETRY_DELAY_MS", 500)
	if generationRetryDelayMs <= 0 {
		generationRetryDelayMs = 500
	}

	queryRewriteModel := strings.TrimSpace(config.GetConfigWithDefault("RAG_QUERY_REWRITE_MODEL", model))
	if queryRewriteModel == "" {
		queryRewriteModel = model
	}

	queryRewriteTimeoutSeconds := config.GetIntConfigWithDefault("RAG_QUERY_REWRITE_TIMEOUT_SECONDS", 5)
	if queryRewriteTimeoutSeconds < 0 {
		queryRewriteTimeoutSeconds = 0
	}

	queryRewriteMaxTokens := config.GetIntConfigWithDefault("RAG_QUERY_REWRITE_MAX_TOKENS", 96)
	if queryRewriteMaxTokens <= 0 {
		queryRewriteMaxTokens = 96
	}
	if queryRewriteMaxTokens > 512 {
		queryRewriteMaxTokens = 512
	}

	return &RagUtils{
		llm:                   llm,
		embedder:              embedder,
		httpClient:            &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		openRouterAPIKey:      openRouterAPIKey,
		model:                 model,
		fallbackModel:         config.GetConfigWithDefault("OPEN_ROUTER_FALLBACK_MODEL", "qwen/qwen3-30b-a3b-instruct-2507"),
		generationMaxRetries:  generationMaxRetries,
		generationRetryDelay:  time.Duration(generationRetryDelayMs) * time.Millisecond,
		rerankEnabled:         config.GetBoolConfigWithDefault("RAG_RERANK_ENABLED", true),
		rerankModel:           config.GetConfigWithDefault("RAG_RERANK_MODEL", "cohere/rerank-4-pro"),
		rerankFallbackModel:   config.GetConfigWithDefault("RAG_RERANK_FALLBACK_MODEL", "cohere/rerank-4-fast"),
		rerankMaxRetries:      rerankMaxRetries,
		rerankRetryDelay:      time.Duration(rerankRetryDelayMs) * time.Millisecond,
		queryRewriteEnabled:   config.GetBoolConfigWithDefault("RAG_QUERY_REWRITE_ENABLED", true),
		queryRewriteModel:     queryRewriteModel,
		queryRewriteTimeout:   time.Duration(queryRewriteTimeoutSeconds) * time.Second,
		queryRewriteMaxTokens: queryRewriteMaxTokens,
	}, nil
}

func (u *RagUtils) StripHTML(text string) string {
	text = htmlTagRegex.ReplaceAllString(text, " ")
	return html.UnescapeString(text)
}

func (u *RagUtils) PrepareChunks(ctx context.Context, text string) ([]string, [][]float32, error) {
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)

	chunks, err := splitter.SplitText(text)
	if err != nil || len(chunks) == 0 {
		return nil, nil, err
	}

	vectors, err := u.embedder.EmbedDocuments(ctx, chunks)
	if err != nil {
		return nil, nil, err
	}

	// Truncate to 1536 dimensions for pgvector compatibility (HNSW index limit is 2000)
	for i := range vectors {
		if len(vectors[i]) > 1536 {
			vectors[i] = vectors[i][:1536]
		}
	}

	return chunks, vectors, nil
}

func (u *RagUtils) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	vectors, err := u.embedder.EmbedDocuments(ctx, []string{query})
	if err != nil || len(vectors) == 0 {
		return nil, err
	}

	vector := vectors[0]
	if len(vector) > 1536 {
		vector = vector[:1536]
	}

	return vector, nil
}

func (u *RagUtils) RewriteSearchQuery(ctx context.Context, question string, history []ChatTurn) (string, error) {
	question = normalizeWhitespace(question)
	if question == "" || !u.queryRewriteEnabled {
		return question, nil
	}

	historyText := buildHistoryForRewrite(history)
	if historyText == "" {
		historyText = "No previous chat turns."
	}

	prompt := fmt.Sprintf(`You rewrite user questions into precise retrieval queries for a history RAG system.

Previous chat turns:
%s

Current user question:
%s

Rules:
- Output exactly one retrieval query inside <query> tags.
- Do not answer the question.
- Do not add names, dates, places, causes, results, or facts unless they are explicitly present in the current question or previous chat turns.
- Use previous chat turns only to resolve clear references such as "đó", "ông ấy", "bà ấy", "sự kiện này", "that event", or "he/she".
- Preserve proper nouns, dates, quoted text, and historical terms exactly when possible.
- If the current question is already clear, return it unchanged.
- If a reference is ambiguous, keep the ambiguous wording instead of guessing.
- Use the same language as the current question.
- No markdown, no bullet points, no explanation.

<query>`, historyText, question)

	rewriteCtx := ctx
	cancel := func() {}
	if u.queryRewriteTimeout > 0 {
		rewriteCtx, cancel = context.WithTimeout(ctx, u.queryRewriteTimeout)
	}
	defer cancel()

	options := []llms.CallOption{
		llms.WithModel(u.queryRewriteModel),
		llms.WithMaxTokens(u.queryRewriteMaxTokens),
		llms.WithTemperature(0),
	}

	rewriteStart := time.Now()
	raw, err := llms.GenerateFromSinglePrompt(rewriteCtx, u.llm, prompt, options...)
	if err != nil {
		return "", err
	}
	log.Info().
		Str("model", u.queryRewriteModel).
		Int("prompt_chars", len(prompt)).
		Int("response_chars", len(raw)).
		Dur("duration", time.Since(rewriteStart)).
		Msg("rag query rewrite succeeded")

	query := normalizeGeneratedQuery(raw)
	if query == "" {
		return question, nil
	}

	return query, nil
}

func (u *RagUtils) RerankDocuments(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if !u.rerankEnabled || len(documents) == 0 {
		return nil, nil
	}
	if topN <= 0 || topN > len(documents) {
		topN = len(documents)
	}

	model := strings.TrimSpace(u.rerankModel)
	if model == "" {
		model = "cohere/rerank-4-pro"
	}

	results, err := u.rerankDocumentsWithModel(ctx, model, query, documents, topN)
	if err == nil {
		return results, nil
	}

	fallbackModel := strings.TrimSpace(u.rerankFallbackModel)
	if fallbackModel == "" || fallbackModel == model {
		return nil, err
	}

	fallbackResults, fallbackErr := u.rerankDocumentsWithModel(ctx, fallbackModel, query, documents, topN)
	if fallbackErr == nil {
		return fallbackResults, nil
	}

	return nil, fmt.Errorf("rerank failed with model %s: %w; fallback model %s failed: %v", model, err, fallbackModel, fallbackErr)
}

func (u *RagUtils) rerankDocumentsWithModel(ctx context.Context, model, query string, documents []string, topN int) ([]RerankResult, error) {
	reqBody := rerankRequest{
		Model:     model,
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= u.rerankMaxRetries; attempt++ {
		attemptStart := time.Now()
		results, retryable, err := u.sendRerankRequest(ctx, payload, documents)
		if err == nil {
			log.Info().
				Str("model", model).
				Int("attempt", attempt+1).
				Int("documents", len(documents)).
				Int("top_n", topN).
				Int("results", len(results)).
				Dur("duration", time.Since(attemptStart)).
				Msg("rag rerank attempt succeeded")
			return results, nil
		}

		log.Warn().
			Err(err).
			Str("model", model).
			Int("attempt", attempt+1).
			Int("documents", len(documents)).
			Int("top_n", topN).
			Bool("retryable", retryable).
			Dur("duration", time.Since(attemptStart)).
			Msg("rag rerank attempt failed")

		lastErr = err
		if !retryable || attempt == u.rerankMaxRetries {
			break
		}

		delay := u.rerankRetryDelay * time.Duration(1<<attempt)
		if delay > 2*time.Second {
			delay = 2 * time.Second
		}
		if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
			return nil, sleepErr
		}
	}

	return nil, fmt.Errorf("rerank failed with model %s: %w", model, lastErr)
}

func (u *RagUtils) sendRerankRequest(ctx context.Context, payload []byte, documents []string) ([]RerankResult, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL+"/rerank", bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+u.openRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := u.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		return nil, true, err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, isRetryableRerankStatus(res.StatusCode), fmt.Errorf("rerank request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, true, err
	}

	var rerankRes rerankResponse
	if err := json.Unmarshal(body, &rerankRes); err != nil {
		return nil, false, err
	}

	results := make([]RerankResult, 0, len(rerankRes.Results))
	for _, item := range rerankRes.Results {
		if item.Index < 0 || item.Index >= len(documents) {
			continue
		}
		results = append(results, RerankResult{
			Index: item.Index,
			Score: item.RelevanceScore,
		})
	}

	return results, false, nil
}

func (u *RagUtils) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	raw, err := u.generateSinglePromptWithRetry(ctx, prompt, u.model)
	if err != nil && u.fallbackModel != "" && u.fallbackModel != u.model {
		raw, err = u.generateSinglePromptWithRetry(ctx, prompt, u.fallbackModel)
	}
	if err != nil {
		return "", err
	}
	return stripThinking(raw), nil
}

func (u *RagUtils) generateSinglePromptWithRetry(ctx context.Context, prompt, model string) (string, error) {
	var lastErr error
	options := make([]llms.CallOption, 0, 1)
	if strings.TrimSpace(model) != "" {
		options = append(options, llms.WithModel(model))
	}

	for attempt := 0; attempt <= u.generationMaxRetries; attempt++ {
		attemptStart := time.Now()
		raw, err := llms.GenerateFromSinglePrompt(ctx, u.llm, prompt, options...)
		if err == nil {
			log.Info().
				Str("model", model).
				Int("attempt", attempt+1).
				Int("prompt_chars", len(prompt)).
				Int("response_chars", len(raw)).
				Dur("duration", time.Since(attemptStart)).
				Msg("rag generation attempt succeeded")
			return raw, nil
		}

		if ctx.Err() != nil {
			log.Warn().
				Err(ctx.Err()).
				Str("model", model).
				Int("attempt", attempt+1).
				Int("prompt_chars", len(prompt)).
				Dur("duration", time.Since(attemptStart)).
				Msg("rag generation attempt canceled")
			return "", ctx.Err()
		}

		log.Warn().
			Err(err).
			Str("model", model).
			Int("attempt", attempt+1).
			Int("prompt_chars", len(prompt)).
			Dur("duration", time.Since(attemptStart)).
			Msg("rag generation attempt failed")

		lastErr = err
		if attempt == u.generationMaxRetries {
			break
		}

		delay := u.generationRetryDelay * time.Duration(1<<attempt)
		if delay > 3*time.Second {
			delay = 3 * time.Second
		}
		if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
			return "", sleepErr
		}
	}

	if strings.TrimSpace(model) == "" {
		return "", lastErr
	}

	return "", fmt.Errorf("generation failed with model %s: %w", model, lastErr)
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

type rerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

func buildHistoryForRewrite(history []ChatTurn) string {
	if len(history) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, turn := range history {
		question := normalizeWhitespace(turn.Question)
		answer := normalizeWhitespace(turn.Answer)
		if question == "" && answer == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("Turn %d\nUser: %s\nAssistant: %s\n", i+1, question, answer))
	}

	return strings.TrimSpace(builder.String())
}

func normalizeGeneratedQuery(raw string) string {
	raw = strings.TrimSpace(raw)

	if query := extractTag(raw, "query"); query != "" {
		return normalizeWhitespace(query)
	}

	raw = strings.TrimSpace(strings.TrimPrefix(raw, "Query:"))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "Retrieval query:"))
	raw = strings.Trim(raw, "\"'`")

	return normalizeWhitespace(raw)
}

func extractTag(raw, tag string) string {
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"

	start := strings.LastIndex(raw, startTag)
	if start == -1 {
		return ""
	}

	content := raw[start+len(startTag):]
	end := strings.Index(content, endTag)
	if end == -1 {
		return strings.TrimSpace(content)
	}

	return strings.TrimSpace(content[:end])
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func isRetryableRerankStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func stripThinking(raw string) string {
	startTag := "<answer>"
	endTag := "</answer>"

	lastStart := strings.LastIndex(raw, startTag)
	if lastStart != -1 {
		content := raw[lastStart+len(startTag):]
		if endIdx := strings.Index(content, endTag); endIdx != -1 {
			return strings.TrimSpace(content[:endIdx])
		}
		return strings.TrimSpace(content)
	}

	if !strings.Contains(raw, "*   ") {
		return strings.TrimSpace(raw)
	}

	lines := strings.Split(raw, "\n")
	answerStart := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "-   ") {
			break
		}
		answerStart = i
	}
	if answerStart < len(lines) {
		answer := strings.TrimSpace(strings.Join(lines[answerStart:], "\n"))
		if answer != "" {
			return answer
		}
	}

	lastLine := lines[len(lines)-1]
	if idx := strings.LastIndex(lastLine, `"`); idx >= 0 && idx < len(lastLine)-1 {
		answer := strings.TrimSpace(lastLine[idx+1:])
		if answer != "" {
			return answer
		}
	}

	return strings.TrimSpace(raw)
}

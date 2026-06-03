package ai

import (
	"context"
	"fmt"
	"history-api/pkg/config"
	"html"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/textsplitter"
)

type RagUtils struct {
	llm      llms.Model
	embedder *embeddings.EmbedderImpl
}

func NewRagUtils() (*RagUtils, error) {
	openRouterAPIKey, err := config.GetConfig("OPEN_ROUTER_API")
	if err != nil {
		return nil, err
	}

	model, err := config.GetConfig("OPEN_ROUTER_MODEL")
	if err != nil {
		model = "qwen/qwen3.5-flash-02-23"
	}

	embeddingModel, err := config.GetConfig("OPEN_ROUTER_EMBEDDING_MODEL")
	if err != nil {
		embeddingModel = "qwen/qwen3-embedding-8b"
	}

	llm, err := openai.New(
		openai.WithToken(openRouterAPIKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel(model),
		openai.WithEmbeddingModel(embeddingModel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init openrouter ai: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to init embedder: %w", err)
	}

	return &RagUtils{
		llm:      llm,
		embedder: embedder,
	}, nil
}

func (u *RagUtils) StripHTML(text string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	text = re.ReplaceAllString(text, " ")
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

func (u *RagUtils) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	raw, err := llms.GenerateFromSinglePrompt(ctx, u.llm, prompt)
	if err != nil {
		return "", err
	}
	return stripThinking(raw), nil
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

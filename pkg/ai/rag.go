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
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/textsplitter"
)

type RagUtils struct {
	llm      llms.Model
	embedder *embeddings.EmbedderImpl
}

func NewRagUtils() (*RagUtils, error) {
	googleAIApiKey, err := config.GetConfig("GOOGLE_AI_API_KEY")
	if err != nil {
		return nil, err
	}

	googleModal, err := config.GetConfig("GOOGLE_AI_MODEL")
	if err != nil {
		googleModal = "gemma-4-26b-a4b-it"
	}

	googleEmbeddingModel, err := config.GetConfig("GOOGLE_AI_EMBEDDING_MODEL")
	if err != nil {
		googleEmbeddingModel = "gemini-embedding-001"
	}

	llm, err := googleai.New(context.Background(),
		googleai.WithAPIKey(googleAIApiKey),
		googleai.WithDefaultModel(googleModal),
		googleai.WithDefaultEmbeddingModel(googleEmbeddingModel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init google ai: %w", err)
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

	return chunks, vectors, nil
}

func (u *RagUtils) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	vectors, err := u.embedder.EmbedDocuments(ctx, []string{query})
	if err != nil || len(vectors) == 0 {
		return nil, err
	}
	return vectors[0], nil
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
	startIdx := strings.Index(raw, startTag)
	endIdx := strings.LastIndex(raw, endTag)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		return strings.TrimSpace(raw[startIdx+len(startTag) : endIdx])
	}

	if startIdx != -1 {
		return strings.TrimSpace(raw[startIdx+len(startTag):])
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

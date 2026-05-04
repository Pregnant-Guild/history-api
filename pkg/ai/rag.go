package ai

import (
	"context"
	"fmt"
	"history-api/pkg/config"
	"html"
	"regexp"

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

	llm, err := googleai.New(context.Background(),
		googleai.WithAPIKey(googleAIApiKey),
		googleai.WithDefaultEmbeddingModel("gemini-embedding-001"),
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
	return llms.GenerateFromSinglePrompt(ctx, u.llm, prompt)
}

package services

import (
	"context"
	"fmt"
	"history-api/internal/repositories"
	"history-api/pkg/ai"
)

type ChatbotService interface {
	Chat(ctx context.Context, projectID *string, question string) (string, error)
}

type chatbotService struct {
	repo     repositories.RagRepository
	ragUtils *ai.RagUtils
}

func NewChatbotService(repo repositories.RagRepository, ragUtils *ai.RagUtils) ChatbotService {
	return &chatbotService{
		repo:     repo,
		ragUtils: ragUtils,
	}
}

func (s *chatbotService) Chat(ctx context.Context, projectID *string, question string) (string, error) {
	qVector, err := s.ragUtils.EmbedQuery(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to embed question: %w", err)
	}
	results, err := s.repo.SearchSimilar(ctx, projectID, qVector, 5, 0.65)
	if err != nil {
		return "", fmt.Errorf("failed to search similar content: %w", err)
	}

	contextStr := ""
	for i, res := range results {
		contextStr += fmt.Sprintf("[Document %d]: %s\n", i+1, res.Content)
	}

	prompt := fmt.Sprintf(`You are a helpful history assistant. Answer the question based ONLY on the provided context.
If the answer is not in the context, say "I don't have enough historical context to answer that."

Context:
%s

Question: %s
Answer:`, contextStr, question)

	return s.ragUtils.GenerateResponse(ctx, prompt)
}

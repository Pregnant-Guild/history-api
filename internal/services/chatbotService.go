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

	var prompt string
	if contextStr == "" {
		prompt = fmt.Sprintf(`You are a friendly history assistant chatbot. The user said: "%s"

Rules:
- If it is a greeting (like "hello", "hi", "xin chào"), respond with a friendly greeting and briefly introduce yourself.
- If it is a history question, say that you don't have relevant documents to answer.
- You MUST wrap your final response inside <answer> tags. Example: <answer>Hello!</answer>
- Do NOT show your reasoning outside or inside the tags if possible, but the final answer MUST be in <answer> tags.`, question)
	} else {
		prompt = fmt.Sprintf(`You are a helpful history assistant. Answer the question using ONLY the provided context.

Rules:
- If the answer is not in the context, say "I don't have enough historical context to answer that."
- You MUST wrap your final response inside <answer> tags. Example: <answer>The capital is...</answer>
- Be concise and direct.

Context:
%s

Question: %s`, contextStr, question)
	}

	return s.ragUtils.GenerateResponse(ctx, prompt)
}

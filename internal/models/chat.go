package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type ConversationEntity struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ModID     *string    `json:"mod_id,omitempty"`
	Status    int16      `json:"status"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type MessageEntity struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	SenderID       string     `json:"sender_id"`
	Content        string     `json:"content"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
}

type ChatbotHistoryEntity struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Question  string     `json:"question"`
	Answer    string     `json:"answer"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

func (c *ChatbotHistoryEntity) ToResponse() *response.ChatbotHistoryDto {
	if c == nil {
		return nil
	}
	return &response.ChatbotHistoryDto{
		ID:        c.ID,
		UserID:    c.UserID,
		Question:  c.Question,
		Answer:    c.Answer,
		CreatedAt: c.CreatedAt,
	}
}

func ChatbotHistoryEntitiesToResponse(entities []*ChatbotHistoryEntity) []*response.ChatbotHistoryDto {
	res := make([]*response.ChatbotHistoryDto, 0, len(entities))
	for _, e := range entities {
		res = append(res, e.ToResponse())
	}
	return res
}

package response

import (
	"time"
)

type ChatbotHistoryDto struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Question  string     `json:"question"`
	Answer    string     `json:"answer"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type GetChatbotHistoryResponse struct {
	Items     []*ChatbotHistoryDto `json:"items"`
	PreCursor string               `json:"pre_cursor,omitempty"`
}

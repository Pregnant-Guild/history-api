package request

type ChatbotDto struct {
	ProjectID *string `json:"project_id"`
	Question  string  `json:"question" validate:"required"`
}

type GetChatbotHistoryDto struct {
	Cursor *string `json:"cursor" query:"cursor" validate:"omitempty,uuid"`
	Limit  int     `json:"limit" query:"limit" validate:"omitempty,min=1,max=100"`
}

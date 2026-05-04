package request

type ChatbotDto struct {
	ProjectID *string `json:"project_id"`
	Question  string  `json:"question" validate:"required"`
}

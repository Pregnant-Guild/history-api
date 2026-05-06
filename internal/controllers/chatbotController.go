package controllers

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"history-api/pkg/validator"
	"time"

	"github.com/gofiber/fiber/v3"
)

type ChatbotController struct {
	chatbotService services.ChatbotService
}

func NewChatbotController(chatbotService services.ChatbotService) *ChatbotController {
	return &ChatbotController{
		chatbotService: chatbotService,
	}
}

// Chat godoc
// @Summary      Ask the AI chatbot
// @Description  Ask a history question based on project context or global knowledge using RAG
// @Tags         Chatbot
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param        request body request.ChatbotDto true "Chatbot query"
// @Success      200 {object} response.CommonResponse{data=string} "Successful response with AI answer"
// @Failure      400 {object} response.CommonResponse "Invalid request"
// @Failure      500 {object} response.CommonResponse "Internal server error"
// @Router       /chatbot/chat [post]
func (cx *ChatbotController) Chat(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dto := &request.ChatbotDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	claims := c.Locals("user").(*response.JWTClaims)

	answer, err := cx.chatbotService.Chat(ctx, claims.UId, dto.ProjectID, dto.Question)
	if err != nil {
		// Trả về lỗi 429 (Too Many Requests) nếu hết lượt dùng
		if err.Error() == "you have reached your daily limit of 10 questions. Please come back tomorrow" {
			return c.Status(fiber.StatusTooManyRequests).JSON(response.CommonResponse{
				Status:  false,
				Message: err.Error(),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   answer,
	})
}

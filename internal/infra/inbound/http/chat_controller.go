package http

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/application/port/inbound"
	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/domain/repository"
)

type ChatController struct {
	getMessageHistory inbound.GetMessageHistoryUseCase
	getChats          inbound.GetChatsByParticipantUseCase
}

func NewChatController(getMessageHistory inbound.GetMessageHistoryUseCase, getChats inbound.GetChatsByParticipantUseCase) *ChatController {
	return &ChatController{getMessageHistory: getMessageHistory, getChats: getChats}
}

func (c *ChatController) Routes(router *gin.Engine) {
	router.GET("/api/v1/chats/messages", c.GetMessageHistory)
	router.GET("/api/v1/chats", c.GetChatsByParticipant)
}

func (c *ChatController) GetChatsByParticipant(ctx *gin.Context) {
	participantID := ctx.Query("userId")
	if participantID == "" {
		ctx.JSON(400, gin.H{"error": "userId is required"})
		return
	}
	chats, err := c.getChats.GetChatsByParticipant(ctx.Request.Context(), participantID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "request failed"})
		return
	}
	ctx.JSON(200, chats)
}

func (c *ChatController) GetMessageHistory(ctx *gin.Context) {
	taskID := ctx.Query("taskId")
	if taskID == "" {
		ctx.JSON(400, gin.H{"error": "taskId is required"})
		return
	}

	messages, err := c.getMessageHistory.GetMessageHistory(ctx.Request.Context(), taskID)
	if err != nil {
		status := 500
		if errors.Is(err, repository.ErrChatNotFound) {
			status = 404
		}
		ctx.JSON(status, gin.H{"error": "request failed"})
		return
	}

	ctx.JSON(200, messages)
}

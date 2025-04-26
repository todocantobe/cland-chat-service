package handler

import (
	"net/http"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	chatUC *usecase.ChatUseCase
}

func NewMessageHandler(chatUC *usecase.ChatUseCase) *MessageHandler {
	return &MessageHandler{chatUC: chatUC}
}

func (h *MessageHandler) GetOfflineMessages(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	ctx := c.Request.Context()

	// Get all messages for user
	allMessages, err := h.chatUC.GetSessionMessages(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	var messages []*entity.Message
	for _, msg := range allMessages {
		if msg.Status == entity.StatusOffline {
			// Update status to delivered
			if err := h.chatUC.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusDelivered); err == nil {
				messages = append(messages, msg)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data": gin.H{
			"messages": messages,
		},
	})
}

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

	// Get user's active sessions
	sessions, err := h.chatUC.sessionRepo.ListActive(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sessions"})
		return
	}

	var messages []*entity.Message
	for _, session := range sessions {
		if session.UserID == userID {
			// Get offline messages for this session
			sessionMessages, err := h.chatUC.messageRepo.GetBySessionID(ctx, session.ID)
			if err != nil {
				continue
			}

			for _, msg := range sessionMessages {
				if msg.Status == entity.StatusOffline {
					// Update status to delivered
					if err := h.chatUC.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusDelivered); err == nil {
						messages = append(messages, msg)
					}
				}
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

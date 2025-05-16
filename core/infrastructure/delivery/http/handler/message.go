package handler

import (
	"net/http"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	messages, err := h.chatUC.GetOfflineMessages(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get offline messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data": gin.H{
			"messages": messages,
		},
	})
}

func (h *MessageHandler) SendChatMessage(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
		Content   string `json:"content"`
		SenderID  string `json:"senderId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()
	// Generate proper IDs
	msgID := "m" + uuid.New().String()
	subSessionID := "ss" + uuid.New().String()

	message := &entity.Message{
		MsgID:       msgID,
		SessionID:   req.SessionID,
		Content:     req.Content,
		Src:         "U:" + req.SenderID,
		Dst:         "S:auto", // Default to bot
		MsgType:     entity.MsgTypeMessage,
		ContentType: entity.ContentTypeText,
		Status:      entity.StatusNew,
		Ts:          entity.StringTimestamp(time.Now().UnixNano() / int64(time.Millisecond)),
		Ext: map[string]interface{}{
			"subSessionId": subSessionID,
		},
	}

	if err := h.chatUC.SendMessage(ctx, message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data": gin.H{
			"msgId":        message.MsgID,
			"sessionId":    message.SessionID,
			"subSessionId": message.SubSessionID,
			"content":      message.Content,
			"src":          message.Src,
			"ts":           message.Ts,
		},
	})
}

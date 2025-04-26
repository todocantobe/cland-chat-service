package handler

import (
	"fmt"
	"net/http"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/domain/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo repository.UserRepository
}

func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) InitUser(c *gin.Context) {
	// Generate user ID and session ID
	userID := "user_" + generateRandomID()
	sessionID := "session_" + generateRandomID()

	// Create user if not exists
	user := &entity.User{
		ID:       userID,
		Username: "guest_" + userID[len(userID)-6:],
		Role:     "customer",
		Status:   "online",
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data": gin.H{
			"userId":    userID,
			"sessionId": sessionID,
			"username":  user.Username,
		},
	})
}

func generateRandomID() string {
	// Simple random ID generation for demo
	// In production, use proper UUID generation
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

package handler

import (
	"net/http"
	"time"

	"cland.org/cland-chat-service/common/utils"
	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/domain/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewUserHandler(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (h *UserHandler) InitUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Check for existing cland-cid cookie
	var clandCID string
	if cookie, err := c.Cookie("cland-cid"); err == nil && utils.IsValidClandCID(cookie) {
		clandCID = cookie
	} else {
		clandCID = utils.GenerateClandCID()
	}

	// Create or update user
	user := &entity.User{
		ID:         clandCID,
		Username:   "guest_" + clandCID[1:7], // Use first 6 chars of UUID
		Role:       "customer",
		Status:     "online",
		LastActive: time.Now(),
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50010010001,
			"msg":  "Failed to initialize user",
			"data": gin.H{
				"error_field":  "user",
				"error_detail": err.Error(),
			},
		})
		return
	}

	// Create new session
	sessionID := utils.GenerateSessionID()
	subSessionID := utils.GenerateSubSessionID()

	session := &entity.Session{
		ID:           sessionID,
		UserID:       clandCID,
		SubSessionID: subSessionID,
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	if err := h.sessionRepo.Create(ctx, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50010010002,
			"msg":  "Failed to create session",
			"data": gin.H{
				"error_field":  "session",
				"error_detail": err.Error(),
			},
		})
		return
	}

	// Set cland-cid cookie
	c.SetCookie(
		"cland-cid",
		clandCID,
		31536000, // 1 year
		"/",
		"",   // TODO: Set domain from config
		true, // Secure
		true, // HttpOnly
	)

	// Generate JWT token
	token, err := utils.GenerateJWT(clandCID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50010010003,
			"msg":  "Failed to generate token",
			"data": gin.H{
				"error_field":  "token",
				"error_detail": err.Error(),
			},
		})
		return
	}

	// Return response matching spec
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Initialization successful",
		"data": gin.H{
			"sessionId":    sessionID,
			"subSessionId": subSessionID,
			"token":        token,
		},
	})
}

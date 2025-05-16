package handler

import (
	"net/http"

	"cland.org/cland-chat-service/core/domain/repository"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	userUC      *usecase.UserUseCase
}

func NewUserHandler(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	userUC *usecase.UserUseCase,
) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		userUC:      userUC,
	}
}

func (h *UserHandler) InitUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Check for existing cland-cid cookie
	var existingCID string
	if cookie, err := c.Cookie("cland-cid"); err == nil {
		existingCID = cookie
	}

	// Call usecase
	res, err := h.userUC.InitUser(ctx, existingCID)
	if err != nil {
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

	// Set cland-cid cookie
	c.SetCookie(
		"cland-cid",
		res.ClandCID,
		31536000, // 1 year
		"/",
		"",   // TODO: Set domain from config
		true, // Secure
		true, // HttpOnly
	)

	// Return response matching spec
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Initialization successful",
		"data": gin.H{
			"sessionId":    res.SessionID,
			"subSessionId": res.SubSessionID,
			"token":        res.Token,
		},
	})
}

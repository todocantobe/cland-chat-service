package handler

import (
	"net/http"

	"cland.org/cland-chat-service/core/domain/repository"
	"cland.org/cland-chat-service/core/infrastructure/delivery/http/response"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
)

// UserResponse represents the response structure for user operations
type UserResponse struct {
	SessionID    string `json:"sessionId"`
	SubSessionID string `json:"subSessionId"`
	Token        string `json:"token"`
}

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

// InitUser initializes a new user session
// @Summary Initialize user session
// @Description Creates a new user session and returns authentication tokens
// @Tags user
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/init [post]
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
		c.JSON(http.StatusInternalServerError, response.Response{
			Code: 50010010001,
			Msg:  "Failed to initialize user",
			Data: gin.H{
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
	c.JSON(http.StatusOK, response.Response{
		Code: 200,
		Msg:  "Initialization successful",
		Data: gin.H{
			"sessionId":    res.SessionID,
			"subSessionId": res.SubSessionID,
			"token":        res.Token,
		},
	})
}

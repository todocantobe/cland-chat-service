package router

import (
	"net/http"
	"sync"

	"cland.org/cland-chat-service/core/infrastructure/delivery/http/handler"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
)

var (
	once   sync.Once
	router *gin.Engine
)

func GetRouter(chatUseCase *usecase.ChatUseCase) *gin.Engine {
	once.Do(func() {
		router = gin.Default()
		setupRoutes(router, chatUseCase)
	})
	return router
}

func setupRoutes(r *gin.Engine, chatUseCase *usecase.ChatUseCase) {
	// CORS middleware
	r.Use(func(c *gin.Context) {
		// Set CORS headers for all responses
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Socket.IO路由
	socketHandler := websocket.NewHandler(chatUseCase)
	r.GET("/socket.io/*any", gin.WrapH(socketHandler.GetServer()))
	r.POST("/socket.io/*any", gin.WrapH(socketHandler.GetServer()))

	// API路由分组
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// User initialization
		userHandler := handler.NewUserHandler(
			chatUseCase.UserRepo,
			chatUseCase.SessionRepo,
		)
		api.POST("/init", userHandler.InitUser)

		// 离线消息
		msgHandler := handler.NewMessageHandler(chatUseCase)
		api.GET("/messages/offline", msgHandler.GetOfflineMessages)
	}
}

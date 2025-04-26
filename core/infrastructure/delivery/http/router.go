package router

import (
	"net/http"
	"sync"

	"cland.org/cland-chat-service/core/infrastructure/config"
	"cland.org/cland-chat-service/core/infrastructure/delivery/http/handler"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	once       sync.Once
	router     *gin.Engine
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			cfg, err := config.Load()
			if err != nil {
				return false
			}
			if cfg.Server.Mode == "debug" {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, allowed := range cfg.Server.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			return false
		},
	}
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

	// WebSocket路由
	/**
	r.GET("/ws", func(c *gin.Context) {
		// Upgrade to WebSocket connection
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to upgrade connection"})
			return
		}

		// Handle connection
		wsHandler := ws.NewHandler(chatUseCase)
		go wsHandler.HandleConnection(conn)
	})
	*/

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

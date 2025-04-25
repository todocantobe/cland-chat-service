package router

import (
	"net/http"
	"sync"

	ws "cland.org/cland-chat-service/core/infrastructure/delivery/websocket"
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
			return true // TODO: 生产环境应限制来源
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
	// WebSocket路由
	r.GET("/ws", func(c *gin.Context) {
		// 认证逻辑
		userID := c.Query("userId")
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 升级为WebSocket连接
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to upgrade connection"})
			return
		}

		// 处理连接
		wsHandler := ws.NewHandler(chatUseCase)
		go wsHandler.HandleConnection(conn, userID)
	})

	// API路由分组
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}

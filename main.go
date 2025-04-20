package main

import (
	"cland.org/cland-chat-service/core/infrastructure/delivery/http/response"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"log"

	"cland.org/cland-chat-service/common/constants"
	"cland.org/cland-chat-service/core/infrastructure/config"
	cland_http "cland.org/cland-chat-service/core/infrastructure/delivery/http"
	cland_ws "cland.org/cland-chat-service/core/infrastructure/delivery/websocket"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.InitConfig(cfg.Log)
	zapLogger := logger.GetLogger()

	// Create Gin router
	httpRouter := cland_http.GetRouter()
	httpRouter.Use(logger.GinRecovery(zapLogger, true))
	httpRouter.Use(logger.GinLogger(zapLogger))

	// Initialize WebSocket manager
	wsManager := connection.NewManager(zapLogger)

	// Register routes
	roote_api := httpRouter.Group("/")
	{
		roote_api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, response.Success(nil))
		})
	}

	api := httpRouter.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	// Create WebSocket server
	wsServer := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.WS.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 升级到 WebSocket
			upgrader := websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				CheckOrigin: func(r *http.Request) bool {
					return true // TODO: 生产环境限制来源
				},
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				zapLogger.Error("Failed to upgrade to WebSocket", zap.Error(err))
				http.Error(w, "Failed to establish WebSocket connection", http.StatusInternalServerError)
				return
			}

			// 获取 userID（从查询参数或认证）
			userID := r.Header.Get(constants.KEY_USER_ID)
			if userID == "" {
				zapLogger.Error("Missing userID for WebSocket connection")
				conn.WriteJSON(cland_ws.WSMessage{
					Code: 40010010001,
					Msg:  "Invalid parameter: user_id is missing",
					Data: map[string]string{"error_field": "user_id"},
				})
				conn.Close()
				return
			}

			// 处理 WebSocket 连接
			wsManager.HandleConnection(conn, userID)
		}),
	}

	// Start servers
	var wg sync.WaitGroup
	wg.Add(2)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: httpRouter,
	}
	go func() {
		defer wg.Done()
		zapLogger.Info("Starting HTTP server", zap.String("address", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start WebSocket server
	go func() {
		defer wg.Done()
		zapLogger.Info("Starting WebSocket server", zap.String("address", wsServer.Addr))
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start WebSocket server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	zapLogger.Info("Shutting down servers...")

	// Shutdown servers
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		zapLogger.Error("Failed to shutdown HTTP server", zap.Error(err))
	}
	if err := wsServer.Shutdown(ctx); err != nil {
		zapLogger.Error("Failed to shutdown WebSocket server", zap.Error(err))
	}

	wg.Wait()
	zapLogger.Info("Servers stopped")
}

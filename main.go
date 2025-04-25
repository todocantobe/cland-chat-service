package main

import (
	"cland.org/cland-chat-service/core/usecase"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"cland.org/cland-chat-service/common/constants"
	"cland.org/cland-chat-service/core/infrastructure/config"
	cland_http "cland.org/cland-chat-service/core/infrastructure/delivery/http"
	cland_ws "cland.org/cland-chat-service/core/infrastructure/delivery/websocket"
	"cland.org/cland-chat-service/core/infrastructure/logger"
	"cland.org/cland-chat-service/core/infrastructure/repository"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		zap.L().Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize logger
	logger.InitConfig(cfg.Log)
	zapLogger := logger.GetLogger()

	// Validate configuration
	if cfg.Server.Port == 0 {
		zapLogger.Fatal("Invalid server port configuration")
	}
	if cfg.WS.Port == 0 {
		zapLogger.Fatal("Invalid WebSocket port configuration")
	}
	if cfg.Server.Port == cfg.WS.Port {
		zapLogger.Fatal("HTTP and WebSocket ports cannot be the same")
	}

	// Initialize repositories
	msgRepo := repository.NewMemoryMessageRepository()
	sessionRepo := repository.NewMemorySessionRepository()
	userRepo := repository.NewMemoryUserRepository()

	// Initialize use cases
	chatUseCase := usecase.NewChatUseCase(
		msgRepo,     // messageRepo
		sessionRepo, // sessionRepo
		userRepo,    // userRepo
	)

	// Create Gin router
	httpRouter := cland_http.GetRouter(chatUseCase)
	httpRouter.Use(logger.GinRecovery(zapLogger, true))
	httpRouter.Use(logger.GinLogger(zapLogger))

	// Initialize WebSocket handler
	wsHandler := cland_ws.NewHandler(chatUseCase)
	
	// Create WebSocket server
	wsServer := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.WS.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 升级到 WebSocket
			upgrader := websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				CheckOrigin: func(r *http.Request) bool {
					origin := r.Header.Get("Origin")
					if cfg.Server.Mode == "production" {
						allowedOrigins := []string{"https://example.com"} // TODO: 从配置读取
						for _, allowed := range allowedOrigins {
							if origin == allowed {
								return true
							}
						}
						return false
					}
					return true // 开发环境允许所有来源
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

			// 处理WebSocket连接
			wsHandler.HandleConnection(conn, userID)
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
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Create main context for the application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	select {
	case sig := <-sigChan:
		zapLogger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		zapLogger.Info("Context cancelled, shutting down")
	}

	zapLogger.Info("Shutting down servers gracefully...")

	// First try graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	var shutdownErr error
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		zapLogger.Error("Failed to shutdown HTTP server gracefully", zap.Error(err))
		shutdownErr = err
	}
	if err := wsServer.Shutdown(shutdownCtx); err != nil {
		zapLogger.Error("Failed to shutdown WebSocket server gracefully", zap.Error(err))
		shutdownErr = err
	}

	wg.Wait()

	if shutdownErr != nil {
		zapLogger.Fatal("Failed to shutdown servers gracefully")
	}
	zapLogger.Info("Servers stopped gracefully")
}

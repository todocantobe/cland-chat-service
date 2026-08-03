// Package websocket 原生 WebSocket 交付层
// 使用 gorilla/websocket 直接升级连接，消息为纯 JSON 文本（无 socket.io / Engine.IO 协议层）
package websocket

import (
	"net/http"
	"sync"
	"time"

	"cland.org/cland-chat-service/core/application"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// 心跳参数
const (
	pongWait   = 60 * time.Second // 读超时：超过该时间未收到 Pong 则断开
	pingPeriod = 20 * time.Second // 服务端 Ping 间隔
	writeWait  = 5 * time.Second  // 写超时
)

// WsServer 原生 WebSocket 服务器
type WsServer struct {
	logger      *zap.Logger
	chatService *application.ChatService
	upgrader    websocket.Upgrader
	connManager *connection.Manager
	sender      dto.MessageSender
	once        sync.Once
}

// NewWsServer 创建 WebSocket 服务器
func NewWsServer(logger *zap.Logger, chatService *application.ChatService) *WsServer {
	return &WsServer{
		logger:      logger,
		chatService: chatService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境建议限制合法域名
			},
		},
	}
}

// InitWsServer 初始化 WebSocket 服务器
func InitWsServer(logger *zap.Logger, chatService *application.ChatService) *WsServer {
	server := NewWsServer(logger, chatService)
	server.once.Do(func() {
		server.connManager = connection.NewManager(logger)
		server.sender = dto.NewMessageSender()
	})
	return server
}

// SetupRoutes 启动独立 WebSocket 服务（端口 8081，路径 /ws）
// 兼容原有调用方式：原生 WS 无需依赖 Gin 路由，独立端口运行
func (s *WsServer) SetupRoutes(_ interface{}) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)

	log := s.logger.Named("websocket")
	log.Info("Starting native WebSocket server on :8081 (path /ws)")
	go func() {
		if err := http.ListenAndServe(":8081", mux); err != nil {
			log.Error("WebSocket server failed to start", zap.Error(err))
		}
	}()
}

// handleWS 处理 WebSocket 握手与连接
func (s *WsServer) handleWS(w http.ResponseWriter, r *http.Request) {
	log := s.logger.Named("websocket")

	// 校验客户端标识
	clandCID := r.URL.Query().Get("cland-cid")
	if clandCID == "" {
		log.Warn("missing cland-cid, rejecting connection")
		http.Error(w, "missing cland-cid", http.StatusBadRequest)
		return
	}

	// 升级为 WebSocket 连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	s.connManager.AddConnection(conn, clandCID)
	s.handleConnection(conn, clandCID)
}

// handleConnection 处理单个连接：心跳保活 + 消息读写循环
func (s *WsServer) handleConnection(conn *websocket.Conn, userID string) {
	log := s.logger.With(zap.String("userID", userID), zap.String("remote", conn.RemoteAddr().String()))

	// 读超时 + Pong 重置
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// 业务消息处理器（复用原生 handler）
	wsHandler := &handler.Handler{
		ChatService:       s.chatService,
		ConnectionManager: s.connManager,
		MessageSender:     s.sender,
	}

	// 连接关闭清理
	defer func() {
		s.connManager.RemoveConnection(userID)
		_ = conn.Close()
		log.Debug("connection closed")
	}()

	// 服务端 Ping 保活协程
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-pingTicker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 消息读取循环：客户端直接发送实体 JSON
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Warn("connection closed unexpectedly", zap.Error(err))
			}
			return
		}
		wsHandler.HandleMessage(conn, string(message))
	}
}

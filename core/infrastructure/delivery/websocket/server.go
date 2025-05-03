package websocket

import (
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"fmt"
	"net/http"
	_ "os"
	_ "strings"
	"sync"
	_ "time"

	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	_ "cland.org/cland-chat-service/core/infrastructure/logger"
	"cland.org/cland-chat-service/core/usecase"
	socketio "github.com/googollee/go-socket.io"
	_ "github.com/googollee/go-socket.io/engineio"
	_ "github.com/googollee/go-socket.io/engineio/transport"
	_ "github.com/googollee/go-socket.io/engineio/transport/polling"
	_ "github.com/googollee/go-socket.io/engineio/transport/websocket"
	"go.uber.org/zap"
)

// WsServer 封装 WebSocket 服务器
type WsServer struct {
	logger      *zap.Logger
	chatUseCase *usecase.ChatUseCase
	server      *socketio.Server
	connManager *connection.Manager
	once        sync.Once
}

// NewWsServer 创建 WebSocket 服务器
func NewWsServer(logger *zap.Logger, chatUseCase *usecase.ChatUseCase) *WsServer {
	return &WsServer{
		logger:      logger,
		chatUseCase: chatUseCase,
	}
}

// InitWsServer 初始化 WebSocket 服务器
func InitWsServer(logger *zap.Logger, chatUseCase *usecase.ChatUseCase) *WsServer {
	server := NewWsServer(logger, chatUseCase)
	server.init()
	return server
}

// init 初始化 WebSocket 配置
func (s *WsServer) init() {
	s.once.Do(func() {
		s.setupWebSocket()
	})
}

// setupWebSocket 配置 Socket.IO 事件
func (s *WsServer) setupWebSocket() {
	log := s.logger.Named("websocket")
	log.Info("Setting up WebSocket server")
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered from panic in setupWebSocket", zap.Any("error", r))
		}
	}()

	// 创建 Socket.IO 服务器
	s.server = socketio.NewServer(nil)

	// 创建连接管理器
	s.connManager = connection.NewManager(s.logger)

	// 创建 WebSocket 处理器
	wsHandler := &handler.Handler{
		Server:            s.server,
		ChatUseCase:       s.chatUseCase,
		ConnectionManager: s.connManager,
	}

	// 设置 Socket.IO 事件处理程序
	s.server.OnConnect("/", func(conn socketio.Conn) error {
		log := log.With(zap.String("conn_id", conn.ID()))
		url := conn.URL()
		query := url.Query()
		clandCID := query.Get("cland-cid")
		log.Info("Client connected", zap.String("cland-cid", clandCID))
		if clandCID == "" {
			log.Warn("Missing cland-cid, rejecting connection")
			return fmt.Errorf("missing cland-cid")
		}
		s.connManager.AddConnection(conn, conn.ID())
		return nil
	})

	s.server.OnEvent("/", "message", func(conn socketio.Conn, msg string) {
		log := log.With(zap.String("conn_id", conn.ID()))
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from panic in OnMessage", zap.Any("error", r))
			}
		}()
		log.Info("Received message", zap.String("message", msg))
		wsHandler.HandleMessage(conn, msg)
	})

	s.server.OnError("/", func(conn socketio.Conn, err error) {
		log := log.With(zap.String("conn_id", conn.ID()))
		log.Error("Socket.IO error", zap.Error(err))
		wsHandler.HandleError(conn, err)
	})

	s.server.OnDisconnect("/", func(conn socketio.Conn, reason string) {
		log := log.With(zap.String("conn_id", conn.ID()))
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from panic in OnDisconnect", zap.Any("error", r))
			}
		}()
		log.Info("Client disconnected", zap.String("reason", reason))
		s.connManager.RemoveConnection(conn.ID())
		//s.server.Remove(conn.ID()) // 清理会话
	})

	go func() {
		err := s.server.Serve()
		if err != nil {
			s.logger.Error("ws serve eror", zap.Error(err))
		}
	}()
	defer func(server *socketio.Server) {
		err := server.Close()
		if err != nil {
			s.logger.Error("ws close error", zap.Error(err))
		}
	}(s.server)

	http.Handle("/socket.io/", s.server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	s.logger.Info("Serving at localhost:8081...")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		s.logger.Error("ws ListenAndServe", zap.Error(err))
		return
	}
}

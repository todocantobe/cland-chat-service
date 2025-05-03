package websocket

import (
	"net/http"
	"sync"

	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WsServer 封装 WebSocket 服务器
type WsServer struct {
	logger      *zap.Logger
	chatUseCase *usecase.ChatUseCase
	upgrader    websocket.Upgrader
	protocol    *SocketIOProtocol
	connManager *connection.Manager
	once        sync.Once
}

// NewWsServer 创建 WebSocket 服务器
func NewWsServer(logger *zap.Logger, chatUseCase *usecase.ChatUseCase) *WsServer {
	return &WsServer{
		logger:      logger,
		chatUseCase: chatUseCase,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
		},
		protocol: NewSocketIOProtocol(),
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

	// 创建连接管理器
	s.connManager = connection.NewManager(s.logger)

	// 创建 HTTP 路由
	http.HandleFunc("/socket.io/", func(w http.ResponseWriter, r *http.Request) {
		// 升级为 WebSocket 连接
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Failed to upgrade connection", zap.Error(err))
			return
		}
		defer conn.Close()

		// 处理连接
		s.handleConnection(conn, r)
	})

	http.Handle("/", http.FileServer(http.Dir("./asset")))
	s.logger.Info("Serving at localhost:8081...")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		s.logger.Error("ws ListenAndServe", zap.Error(err))
	}
}

// handleConnection 处理 WebSocket 连接
func (s *WsServer) handleConnection(conn *websocket.Conn, r *http.Request) {
	log := s.logger.With(zap.String("remote_addr", conn.RemoteAddr().String()))

	// 获取连接参数
	clandCID := r.URL.Query().Get("cland-cid")
	if clandCID == "" {
		log.Warn("Missing cland-cid, rejecting connection")
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "missing cland-cid"))
		return
	}

	// 创建 WebSocket 处理器
	wsHandler := &handler.Handler{
		ChatUseCase:       s.chatUseCase,
		ConnectionManager: s.connManager,
	}

	// 添加连接到管理器
	s.connManager.AddConnection(conn, clandCID)

	// 发送连接成功确认
	s.protocol.SendEvent(conn, "connection_success", map[string]string{
		"message":  "Connected successfully",
		"clandCID": clandCID,
	}, "/")

	// 处理消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket read error", zap.Error(err))
			}
			s.connManager.RemoveConnection(clandCID)
			return
		}

		// 处理 Socket.IO 协议消息
		msg, err := s.protocol.parseMessage(message)
		if err != nil {
			log.Error("Failed to parse message", zap.Error(err))
			continue
		}

		switch msg.Type {
		case PacketTypeEvent:
			event, args, err := s.protocol.parseEventData(msg)
			if err != nil {
				log.Error("Failed to parse event data", zap.Error(err))
				continue
			}

			if event == "message" && len(args) > 0 {
				if msgStr, ok := args[0].(string); ok {
					wsHandler.HandleMessage(conn, msgStr)
				}
			}
		case PacketTypeDisconnect:
			s.connManager.RemoveConnection(clandCID)
			return
		}
	}
}

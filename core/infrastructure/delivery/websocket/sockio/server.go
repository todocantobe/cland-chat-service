package sockio

import (
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// WsServer 封装 WebSocket 服务器
type WsServer struct {
	logger      *zap.Logger
	chatUseCase *usecase.ChatUseCase
	upgrader    websocket.Upgrader
	protocol    *EngineIOProtocol
	connManager *connection.Manager
	once        sync.Once
}

// NewWsServer creates a new WebSocket server
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
		protocol: NewEngineIOProtocol(),
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
		// 检查是否是 Socket.IO 握手请求
		if r.URL.Query().Get("EIO") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Handle polling transport
		if r.Method == "GET" && r.URL.Query().Get("transport") == "polling" {
			sid := generateSessionID() // Implement this function
			if err := s.protocol.SendHandshake(w, sid); err != nil {
				log.Error("Failed to send handshake", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			return
		}

		// Handle WebSocket transport - only upgrade if Upgrade header is present
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			conn, err := s.upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error("Failed to upgrade connection", zap.Error(err))
				return
			}

			// Send handshake ack
			sid := generateSessionID() // Implement this function
			if err := s.protocol.SendPacket(conn, PacketTypeOpen, map[string]interface{}{
				"sid":          sid,
				"upgrades":     []string{"websocket"},
				"pingInterval": 25000,
				"pingTimeout":  5000,
			}); err != nil {
				log.Error("Failed to send handshake ack", zap.Error(err))
				conn.Close()
				return
			}

			// Handle connection
			if clandCid, err := s.handleConnection(conn, r); err != nil {
				log.Error("Failed to send handshake ack", zap.Error(err))
				conn.Close()
				return
			} else {
				s.handle0(conn, clandCid)
			}
			return
		}

		// Continue with normal HTTP handling if not WebSocket upgrade
		w.WriteHeader(http.StatusBadRequest)
	})

	http.Handle("/", http.FileServer(http.Dir("./asset")))
	s.logger.Info("Serving at localhost:8081...")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		s.logger.Error("ws ListenAndServe", zap.Error(err))
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return "sess_" + time.Now().Format("20060102150405") + "_" + randString(10)
}

// randString generates a random string of given length
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// handleConnection handles WebSocket connections
func (s *WsServer) handleConnection(conn *websocket.Conn, r *http.Request) (string, error) {
	log := s.logger.With(zap.String("remote_addr", conn.RemoteAddr().String()))

	// Get connection parameters
	clandCID := r.URL.Query().Get("cland-cid")
	if clandCID == "" {
		log.Warn("Missing cland-cid, rejecting connection")
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "missing cland-cid"))
		return "", errors.New("miss cid")
	}
	// Add connection to manager
	s.connManager.AddConnection(conn, clandCID)

	return clandCID, nil

}

func (s *WsServer) handle0(conn *websocket.Conn, clandCID string) {
	log := s.logger.With(zap.String("remote_addr", conn.RemoteAddr().String()))
	// Create WebSocket handler
	wsHandler := &handler.Handler{
		ChatUseCase:       s.chatUseCase,
		ConnectionManager: s.connManager,
	}

	// Handle messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket read error", zap.Error(err))
			}
			s.connManager.RemoveConnection(clandCID)
			return
		}

		// Parse Engine.IO packet
		packetType, payload, err := s.protocol.ParsePacket(message)
		if err != nil {
			log.Error("Failed to parse packet", zap.Error(err))
			continue
		}

		switch packetType {
		case PacketTypeMessage:
			// Parse Socket.IO packet
			sioType, namespace, sioPayload, err := s.protocol.ParseSocketIOPacket(payload)
			if err != nil {
				log.Error("Failed to parse Socket.IO packet", zap.Error(err))
				continue
			}

			switch sioType {
			case SocketIOPacketConnect:
				// Handle namespace connection
				log.Info("Client connected to namespace", zap.String("namespace", namespace))
				// Build and send connection ack
				ackPacket, err := s.protocol.BuildSocketIOPacket(SocketIOPacketConnect, namespace, map[string]string{
					"sid": generateSessionID(),
				})
				if err != nil {
					log.Error("Failed to build connect ack", zap.Error(err))
					continue
				}
				if err := s.protocol.SendPacket(conn, PacketTypeMessage, ackPacket); err != nil {
					log.Error("Failed to send connect ack", zap.Error(err))
				}
			default:
				// Handle other Socket.IO messages
				wsHandler.HandleMessage(conn, string(sioPayload))
			}
		case PacketTypePing:
			// Respond to ping
			if err := s.protocol.SendPacket(conn, PacketTypePong, nil); err != nil {
				log.Error("Failed to send pong", zap.Error(err))
			}
		case PacketTypeClose:
			s.connManager.RemoveConnection(clandCID)
			return
		}
	}
}

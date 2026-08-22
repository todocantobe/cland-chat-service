package sockio

import (
	"encoding/json"
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

	// 创建连接管理器（连接注册唯一来源）
	s.connManager = connection.NewManager(s.logger)

	// 创建 HTTP 路由
	http.HandleFunc("/socket.io/", func(w http.ResponseWriter, r *http.Request) {
		// 检查是否是 Socket.IO 握手请求
		if r.URL.Query().Get("EIO") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 极简网关仅支持 websocket transport：
		// polling 明确拒绝（标准客户端收到错误后自动回退 websocket）
		transport := r.URL.Query().Get("transport")
		if transport != "websocket" {
			log.Warn("Unsupported transport", zap.String("transport", transport))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":400,"message":"cland-chat-service only supports websocket transport, client should use transports:['websocket']"}`))
			return
		}

		// 仅处理 WebSocket 升级请求
		if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 认证优先：cland-cid 缺失直接 HTTP 拒绝，避免"先发 open 再关闭"的顺序瑕疵
		clandCID := r.URL.Query().Get("cland-cid")
		if clandCID == "" {
			log.Warn("Missing cland-cid, rejecting connection", zap.String("remote_addr", r.RemoteAddr))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":40010010001,"message":"missing cland-cid"}`))
			return
		}

		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Failed to upgrade connection", zap.Error(err))
			return
		}

		// 先注册连接（唯一注册表），再发 Engine.IO open，最后进入事件循环
		s.connManager.AddConnection(conn, clandCID)
		if err := s.protocol.SendPacket(conn, PacketTypeOpen, map[string]interface{}{
			"sid":          generateSessionID(),
			"upgrades":     []string{},
			"pingInterval": 25000,
			"pingTimeout":  5000,
		}); err != nil {
			log.Error("Failed to send open packet", zap.Error(err))
			s.connManager.RemoveConnection(clandCID)
			conn.Close()
			return
		}

		s.serveConnection(conn, clandCID)
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

// serveConnection 事件循环：Engine.IO 帧处理 + Socket.IO 包分发 + 服务端心跳
func (s *WsServer) serveConnection(conn *websocket.Conn, clandCID string) {
	log := s.logger.With(zap.String("cid", clandCID), zap.String("remote_addr", conn.RemoteAddr().String()))

	messageSender := NewSocketIOMessageSender(s.protocol, s.logger)
	wsHandler := &handler.Handler{
		ChatUseCase:       s.chatUseCase,
		ConnectionManager: s.connManager,
		MessageSender:     messageSender,
	}

	// gorilla/websocket 同一时刻只允许一个 writer，用互斥锁串行化心跳与应答写
	var writeMu sync.Mutex
	send := func(packetType string, data interface{}) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return s.protocol.SendPacket(conn, packetType, data)
	}

	const pingInterval = 25 * time.Second // 与 open 包 pingInterval 一致
	const pingTimeout = 20 * time.Second  // 与 open 包 pingTimeout 一致

	// 心跳：服务端周期性发 ping（Engine.IO v4 由服务端发起），客户端须回 pong
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := send(PacketTypePing, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 读循环：每次读设置截止时间，客户端超时未活跃即判定离线
	for {
		conn.SetReadDeadline(time.Now().Add(pingInterval + pingTimeout))
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket read error", zap.Error(err))
			}
			s.connManager.RemoveConnection(clandCID)
			conn.Close()
			return
		}

		// Parse Engine.IO packet
		packetType, payload, err := s.protocol.ParsePacket(message)
		if err != nil {
			log.Error("Failed to parse packet", zap.Error(err))
			continue
		}

		switch packetType {
		case PacketTypePing:
			// 客户端探测/心跳：回 pong 并回显载荷（2probe -> 3probe）
			s.connManager.UpdateLastActive(clandCID)
			if err := send(PacketTypePong, string(payload)); err != nil {
				log.Error("Failed to send pong", zap.Error(err))
			}
		case PacketTypePong:
			// 心跳应答
			s.connManager.UpdateLastActive(clandCID)
		case PacketTypeUpgrade:
			// polling->websocket 升级确认：回 noop
			if err := send(PacketTypeNoop, nil); err != nil {
				log.Error("Failed to send noop", zap.Error(err))
			}
		case PacketTypeNoop:
			// 忽略
		case PacketTypeClose:
			s.connManager.RemoveConnection(clandCID)
			conn.Close()
			return
		case PacketTypeMessage:
			s.connManager.UpdateLastActive(clandCID)
			s.handleSocketIOMessage(conn, clandCID, payload, wsHandler, send)
		}
	}
}

// handleSocketIOMessage Socket.IO 包分发：connect 确认 / 事件路由（message/join/leave）/ disconnect
func (s *WsServer) handleSocketIOMessage(conn *websocket.Conn, clandCID string, payload []byte, wsHandler *handler.Handler, send func(string, interface{}) error) {
	log := s.logger.With(zap.String("cid", clandCID))

	sioType, namespace, sioPayload, _, err := s.protocol.ParseSocketIOPacket(payload)
	if err != nil {
		log.Error("Failed to parse Socket.IO packet", zap.Error(err), zap.ByteString("payload", payload))
		return
	}

	switch sioType {
	case SocketIOPacketConnect:
		// 命名空间连接确认：40{"sid":"..."}（默认命名空间）
		log.Info("Client connected to namespace", zap.String("namespace", namespace))
		ackPacket, err := s.protocol.BuildSocketIOPacket(SocketIOPacketConnect, namespace, map[string]string{
			"sid": generateSessionID(),
		})
		if err != nil {
			log.Error("Failed to build connect ack", zap.Error(err))
			return
		}
		if err := send(PacketTypeMessage, ackPacket); err != nil {
			log.Error("Failed to send connect ack", zap.Error(err))
		}
	case SocketIOPacketDisconnect:
		// 客户端主动断开
		s.connManager.RemoveConnection(clandCID)
		conn.Close()
	case SocketIOPacketEvent, SocketIOPacketBinaryEvent:
		eventName, eventData, err := s.protocol.ParseEventPayload(sioPayload)
		if err != nil {
			log.Error("Failed to parse event payload", zap.Error(err), zap.ByteString("payload", sioPayload))
			return
		}

		// 事件分发：message -> 业务消息；join/leave -> 房间管理（修复前零调用）
		switch eventName {
		case "message":
			wsHandler.HandleMessage(conn, string(eventData))
		case "join":
			roomID, err := parseEventString(eventData)
			if err != nil {
				log.Error("Failed to parse join room id", zap.Error(err))
				return
			}
			s.connManager.JoinRoom(clandCID, roomID)
		case "leave":
			roomID, err := parseEventString(eventData)
			if err != nil {
				log.Error("Failed to parse leave room id", zap.Error(err))
				return
			}
			s.connManager.LeaveRoom(clandCID, roomID)
		default:
			log.Debug("Unhandled event", zap.String("event", eventName))
		}
	case SocketIOPacketAck, SocketIOPacketBinaryAck:
		// 本网关不发起 ack，客户端 ack 忽略
		log.Debug("Ignoring ack packet")
	case SocketIOPacketConnectError:
		log.Warn("Client connect error", zap.ByteString("payload", sioPayload))
	default:
		log.Debug("Unhandled Socket.IO packet type", zap.String("type", sioType))
	}
}

// parseEventString 解析事件数据中的字符串参数（如 ["join","roomA"] 的 "roomA"）
func parseEventString(raw []byte) (string, error) {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", errors.New("event data must be a string")
	}
	return v, nil
}

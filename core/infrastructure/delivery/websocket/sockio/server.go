package sockio

import (
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"cland.org/cland-chat-service/core/application"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// WsServer 封装 WebSocket 服务器
type WsServer struct {
	logger      *zap.Logger
	chatService *application.ChatService
	upgrader    websocket.Upgrader
	protocol    *EngineIOProtocol
	connManager *connection.Manager
	once        sync.Once
	mu          sync.Mutex // 防止 Session ID 重复的锁
}

// NewWsServer creates a new WebSocket server
func NewWsServer(logger *zap.Logger, chatService *application.ChatService) *WsServer {
	return &WsServer{
		logger:      logger,
		chatService: chatService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (生产环境建议限制合法域名)
			},
		},
		protocol: NewEngineIOProtocol(),
	}
}

// InitWsServer 初始化 WebSocket 服务器
func InitWsServer(logger *zap.Logger, chatService *application.ChatService) *WsServer {
	server := NewWsServer(logger, chatService)
	server.init()
	return server
}

// init 初始化 WebSocket 配置
func (s *WsServer) init() {
	s.once.Do(func() {
		// 创建连接管理器（确保只初始化一次）
		s.connManager = connection.NewManager(s.logger)
	})
}

// SetupRoutes 设置 WebSocket 路由
func (s *WsServer) SetupRoutes(router interface{}) {
	// For separate WebSocket server on port 8081, we don't need to integrate with Gin
	s.setupWebSocket()
}

// setupWebSocket 配置 Socket.IO 事件
func (s *WsServer) setupWebSocket() {
	log := s.logger.Named("websocket")
	log.Info("Setting up WebSocket server on port 8081")
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered from panic in setupWebSocket", zap.Any("error", r))
		}
	}()

	// 确保连接管理器只初始化一次（修复重复创建问题）
	if s.connManager == nil {
		s.init()
	}

	// 创建 HTTP 路由
	mux := http.NewServeMux()
	mux.HandleFunc("/socket.io/", func(w http.ResponseWriter, r *http.Request) {
		// 检查是否是 Socket.IO 握手请求
		if r.URL.Query().Get("EIO") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Handle polling transport
		if r.Method == "GET" && r.URL.Query().Get("transport") == "polling" {
			sid := s.generateSessionID()
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

			// Send Engine.IO v4 handshake immediately after WebSocket upgrade
			sid := s.generateSessionID()
			if err := s.protocol.SendPacket(conn, PacketTypeOpen, map[string]interface{}{
				"sid":          sid,
				"upgrades":     []string{},
				"pingInterval": 25000, // 客户端每 25 秒发一次 Ping
				"pingTimeout":  5000,  // 5 秒内未收到 Pong 则断开
			}); err != nil {
				log.Error("Failed to send handshake", zap.Error(err))
				_ = conn.Close() // 出错时确保关闭连接
				return
			}

			// Handle the connection
			go s.handle0(conn, r, sid) // 启动 goroutine 处理连接（避免阻塞主线程）
			return
		}

		// Continue with normal HTTP handling if not WebSocket upgrade
		w.WriteHeader(http.StatusBadRequest)
	})

	log.Info("Starting WebSocket server on :8081...")
	err := http.ListenAndServe(":8081", mux)
	if err != nil {
		log.Error("WebSocket server failed to start", zap.Error(err))
	}
}

// generateSessionID 生成唯一 Session ID（加锁防止重复）
func (s *WsServer) generateSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return "sess_" + time.Now().Format("20060102150405.000000") + "_" + s.randString(16) // 毫秒级时间+16位随机串
}

// randString 生成随机字符串
func (s *WsServer) randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// handle0 处理单个 WebSocket 连接（优化心跳和错误处理）
func (s *WsServer) handle0(conn *websocket.Conn, r *http.Request, sid string) {
	log := s.logger.With(zap.String("remote_addr", conn.RemoteAddr().String()), zap.String("sid", sid))
	clandCID := r.URL.Query().Get("cland-cid")

	// 延迟清理：确保连接关闭时移除管理器中的记录并关闭连接
	defer func() {
		log.Debug("Closing WebSocket connection")
		s.connManager.RemoveConnection(clandCID)
		_ = conn.Close() // 主动关闭连接，避免资源泄露
	}()

	// 校验 cland-cid
	if clandCID == "" {
		log.Warn("Missing cland-cid, rejecting connection")
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "missing cland-cid"))
		return
	}

	// 添加连接到管理器（初始化最后活动时间）
	s.connManager.AddConnection(conn, clandCID)
	s.connManager.UpdateLastActive(clandCID) // 确保初始状态有活动时间

	// 创建处理器
	messageSender := NewSocketIOMessageSender(s.protocol, s.logger)
	wsHandler := &handler.Handler{
		ChatService:       s.chatService,
		ConnectionManager: s.connManager,
		MessageSender:     messageSender,
	}
	
	// 创建协议处理器
	protocolHandler := NewProtocolHandler(s.logger, s.protocol, wsHandler)

	// 配置读超时：基于 Engine.IO 握手约定的 pingTimeout（5秒）+ 缓冲1秒
	readTimeout := 6 * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))

	// 心跳响应回调：收到 Pong 时更新读超时和活动时间
	conn.SetPongHandler(func(string) error {
		s.connManager.UpdateLastActive(clandCID)
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout)) // 重置读超时
		return nil
	})

	// 服务端主动 Ping 协程（每 20 秒发一次，确保中间件不主动断开）
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()

	// 连接超时清理协程（每 30 秒检查一次）
	timeoutTicker := time.NewTicker(30 * time.Second)
	defer timeoutTicker.Stop()

	// 消息处理循环
	for {
		select {
		case <-pingTicker.C:
			// 服务端主动发送 Ping（维持连接活性）
			_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second)) // Ping 写入超时
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("Failed to send ping", zap.Error(err))
				return
			}
			_ = conn.SetWriteDeadline(time.Time{}) // 重置写入超时

		case <-timeoutTicker.C:
			// 检查当前连接是否超时（超过 60 秒无活动）
			if s.connManager.IsConnectionTimeout(clandCID, 60*time.Second) {
				log.Info("Connection timed out (no activity)")
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1008, "connection timeout"))
				return
			}

		default:
			// 读取客户端消息（非阻塞，避免占用 CPU）
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			_, message, err := conn.ReadMessage()
			_ = conn.SetReadDeadline(time.Now().Add(readTimeout)) // 重置读超时

			if err != nil {
				// 处理 WebSocket 关闭错误
				if closeErr, ok := err.(*websocket.CloseError); ok {
					switch closeErr.Code {
					case 1000, 1001, 1005: // 正常关闭：1000(正常)、1001(客户端离开)、1005(无状态)
						log.Debug("WebSocket normal close", zap.Int("code", closeErr.Code), zap.String("text", closeErr.Text))
					default: // 异常关闭
						log.Error("WebSocket abnormal close", zap.Int("code", closeErr.Code), zap.String("text", closeErr.Text))
					}
					return
				}

				// 忽略读超时错误（非阻塞读取时正常）
				if strings.Contains(err.Error(), "i/o timeout") {
					continue
				}

				// 其他异常错误
				log.Error("WebSocket read error", zap.Error(err))
				return
			}

			// 解析 Engine.IO 数据包
			packetType, payload, err := s.protocol.ParsePacket(message)
			if err != nil {
				log.Error("Failed to parse Engine.IO packet", zap.Error(err), zap.ByteString("message", message))
				continue
			}

			// 使用协议处理器处理数据包
			if err := protocolHandler.HandleEngineIOPacket(conn, packetType, payload, clandCID); err != nil {
				log.Info("Connection terminated by protocol handler", zap.Error(err))
				return
			}
		}
	}
}

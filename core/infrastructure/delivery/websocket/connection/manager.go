package connection

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Manager WebSocket连接管理器
type Manager struct {
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	log         *zap.Logger
}

// NewManager 创建WebSocket连接管理器
func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		connections: make(map[string]*websocket.Conn),
		log:         log,
	}
}

// HandleConnection 处理WebSocket连接
func (m *Manager) HandleConnection(conn *websocket.Conn, userID string) {
	// 验证用户ID
	if userID == "" {
		m.log.Error("Empty user ID provided")
		conn.Close()
		return
	}

	// 存储连接
	m.mu.Lock()
	m.connections[userID] = conn
	m.mu.Unlock()

	m.log.Info("New WebSocket connection", zap.String("userID", userID))

	// 处理消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.log.Error("WebSocket read error", zap.Error(err), zap.String("userID", userID))
			}
			break
		}

		// 处理消息
		var msg struct {
			To      string `json:"to"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			m.log.Error("Failed to unmarshal message", zap.Error(err), zap.String("userID", userID))
			continue
		}

		// 发送消息
		if err := m.SendMessage(msg.To, message); err != nil {
			m.log.Error("Failed to send message", zap.Error(err), zap.String("to", msg.To))
		}
	}

	// 清理连接
	m.mu.Lock()
	delete(m.connections, userID)
	m.mu.Unlock()

	m.log.Info("WebSocket connection closed", zap.String("userID", userID))
	conn.Close()
}

// SendMessage 发送消息到指定用户
func (m *Manager) SendMessage(userID string, message interface{}) error {
	m.mu.RLock()
	conn, ok := m.connections[userID]
	m.mu.RUnlock()

	if !ok {
		m.log.Warn("User not connected", zap.String("userID", userID))
		return fmt.Errorf("user %s not connected", userID)
	}

	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		m.log.Error("Failed to marshal message", zap.Error(err))
		return err
	}

	// 发送消息
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		m.log.Error("Failed to write message", zap.Error(err), zap.String("userID", userID))
		return err
	}

	m.log.Info("Message sent", zap.String("to", userID))
	return nil
}

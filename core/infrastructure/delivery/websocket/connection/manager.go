package connection

import (
	"encoding/json"
	"fmt"
	"sync"

	socketio "github.com/googollee/go-socket.io"
	"go.uber.org/zap"
)

// Manager Socket.IO连接管理器
type Manager struct {
	connections map[string]socketio.Conn
	mu          sync.RWMutex
	log         *zap.Logger
}

// NewManager 创建Socket.IO连接管理器
func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		connections: make(map[string]socketio.Conn),
		log:         log,
	}
}

// AddConnection 添加连接
func (m *Manager) AddConnection(conn socketio.Conn, userID string) {
	if userID == "" {
		m.log.Error("Empty user ID provided")
		return
	}

	m.mu.Lock()
	m.connections[userID] = conn
	m.mu.Unlock()

	m.log.Info("New Socket.IO connection", zap.String("userID", userID))
}

// RemoveConnection 移除连接
func (m *Manager) RemoveConnection(userID string) {
	m.mu.Lock()
	delete(m.connections, userID)
	m.mu.Unlock()

	m.log.Info("Socket.IO connection removed", zap.String("userID", userID))
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
	conn.Emit("message", data) // Emit方法不返回错误

	m.log.Info("Message sent", zap.String("to", userID))
	return nil
}

// BroadcastMessage 广播消息
func (m *Manager) BroadcastMessage(message interface{}, userIDs []string) error {
	data, err := json.Marshal(message)
	if err != nil {
		m.log.Error("Failed to marshal message", zap.Error(err))
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, userID := range userIDs {
		if conn, ok := m.connections[userID]; ok {
			conn.Emit("message", data) // Emit方法不返回错误
		}
	}
	return nil
}

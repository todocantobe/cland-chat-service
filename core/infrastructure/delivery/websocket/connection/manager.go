package connection

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Manager Socket.IO连接管理器
type Manager struct {
	connections map[string]*websocket.Conn // userID -> connection
	rooms       map[string]map[string]bool // roomID -> userIDs
	mu          sync.RWMutex
	log         *zap.Logger
}

// NewManager 创建Socket.IO连接管理器
func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		connections: make(map[string]*websocket.Conn),
		rooms:       make(map[string]map[string]bool),
		log:         log,
	}
}

// AddConnection 添加连接
func (m *Manager) AddConnection(conn *websocket.Conn, userID string) {
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
	return conn.WriteMessage(websocket.TextMessage, data)
}

// JoinRoom 加入房间
func (m *Manager) JoinRoom(userID, roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rooms[roomID]; !exists {
		m.rooms[roomID] = make(map[string]bool)
	}
	m.rooms[roomID][userID] = true
	m.log.Info("User joined room", zap.String("userID", userID), zap.String("roomID", roomID))
}

// LeaveRoom 离开房间
func (m *Manager) LeaveRoom(userID, roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if users, exists := m.rooms[roomID]; exists {
		delete(users, userID)
		if len(users) == 0 {
			delete(m.rooms, roomID)
		}
		m.log.Info("User left room", zap.String("userID", userID), zap.String("roomID", roomID))
	}
}

// BroadcastMessage 广播消息给多个用户
func (m *Manager) BroadcastMessage(message interface{}, userIDs []string) error {
	return m.sendToUsers(message, userIDs)
}

// BroadcastToRoom 广播消息到房间
func (m *Manager) BroadcastToRoom(message interface{}, roomID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if users, exists := m.rooms[roomID]; exists {
		userIDs := make([]string, 0, len(users))
		for userID := range users {
			userIDs = append(userIDs, userID)
		}
		return m.sendToUsers(message, userIDs)
	}
	return nil
}

// sendToUsers 内部方法：发送消息给多个用户
func (m *Manager) sendToUsers(message interface{}, userIDs []string) error {
	data, err := json.Marshal(message)
	if err != nil {
		m.log.Error("Failed to marshal message", zap.Error(err))
		return err
	}

	for _, userID := range userIDs {
		if conn, ok := m.connections[userID]; ok {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return err
			}
		}
	}
	return nil
}

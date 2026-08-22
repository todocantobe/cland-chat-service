package connection

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Manager Socket.IO连接管理器
type Manager struct {
	connections map[string]*websocket.Conn // userID -> connection
	lastActive  map[string]time.Time       // userID -> last active time
	rooms       map[string]map[string]bool // roomID -> userIDs
	mu          sync.RWMutex
	log         *zap.Logger
}

// NewManager 创建Socket.IO连接管理器
func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		connections: make(map[string]*websocket.Conn),
		lastActive:  make(map[string]time.Time),
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
	m.lastActive[userID] = time.Now()
	m.mu.Unlock()

	m.log.Info("New Socket.IO connection", zap.String("userID", userID))
}

// RemoveConnection 移除连接
func (m *Manager) RemoveConnection(userID string) {
	m.mu.Lock()
	delete(m.connections, userID)
	delete(m.lastActive, userID)
	// 同时从所有房间移除（避免房间成员残留）
	for roomID, users := range m.rooms {
		if _, ok := users[userID]; ok {
			delete(users, userID)
			if len(users) == 0 {
				delete(m.rooms, roomID)
			}
		}
	}
	m.mu.Unlock()

	m.log.Info("Socket.IO connection removed", zap.String("userID", userID))
}

// UpdateLastActive 更新最后活跃时间
func (m *Manager) UpdateLastActive(userID string) {
	m.mu.Lock()
	if _, exists := m.connections[userID]; exists {
		m.lastActive[userID] = time.Now()
	}
	m.mu.Unlock()
}

// CheckTimeoutConnections 检查超时连接并关闭
func (m *Manager) CheckTimeoutConnections(timeout time.Duration) []string {
	var timedOut []string
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, lastActive := range m.lastActive {
		if now.Sub(lastActive) > timeout {
			if conn, exists := m.connections[userID]; exists {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "connection timeout"))
				conn.Close()
				delete(m.connections, userID)
				delete(m.lastActive, userID)
				timedOut = append(timedOut, userID)
			}
		}
	}

	return timedOut
}

// GetConnection 获取用户连接（唯一连接注册表入口）
func (m *Manager) GetConnection(userID string) (*websocket.Conn, bool) {
	m.mu.RLock()
	conn, ok := m.connections[userID]
	m.mu.RUnlock()
	return conn, ok
}

// GetRoomConnections 获取房间内所有在线连接
func (m *Manager) GetRoomConnections(roomID string) []*websocket.Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var conns []*websocket.Conn
	if users, exists := m.rooms[roomID]; exists {
		for userID := range users {
			if conn, ok := m.connections[userID]; ok {
				conns = append(conns, conn)
			}
		}
	}
	return conns
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

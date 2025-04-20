package entity

import "time"

// Message 消息实体
type Message struct {
	ID        uint      `json:"id"`
	SessionID string    `json:"session_id"`
	FromUser  string    `json:"from_user"`
	ToUser    string    `json:"to_user"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // text, image, file
	Status    string    `json:"status"` // sent, delivered, read
	CreatedAt time.Time `json:"created_at"`
}

// Session 会话实体
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AgentID   string    `json:"agent_id"`
	Status    string    `json:"status"` // active, closed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User 用户实体
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"` // customer, agent, admin
	Status   string `json:"status"` // online, offline, busy
} 
package websocket

import (
	"errors"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
)

// WSMessage WebSocket通用消息结构
type WSMessage struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// ChatMessage 聊天消息DTO
type ChatMessage struct {
	entity.Message
}

// FromEntity 从实体转换
func FromEntity(msg entity.Message) ChatMessage {
	return ChatMessage{Message: msg}
}

// ToEntity 转换为实体
func (m ChatMessage) ToEntity() entity.Message {
	return m.Message
}

// Validate 验证消息有效性
func (m ChatMessage) Validate() error {
	if m.SessionID == "" {
		return errors.New("sessionId is required")
	}
	if m.MsgID == "" {
		return errors.New("msgId is required")
	}
	if m.Src == "" || m.Dst == "" {
		return errors.New("src and dst are required")
	}
	if m.Content == "" && m.MsgType != entity.MsgTypeAck {
		return errors.New("content is required for non-ack messages")
	}
	return nil
}

// UpdateStatus 更新消息状态
func (m *ChatMessage) UpdateStatus(newStatus uint8) {
	m.Status = newStatus
	m.Ts = time.Now().Format(time.RFC3339)
}

// IsValidTransition 检查状态转换是否有效
func (m ChatMessage) IsValidTransition(newStatus uint8) bool {
	switch m.Status {
	case entity.StatusNew:
		return newStatus == entity.StatusSent || newStatus == entity.StatusOffline
	case entity.StatusSent:
		return newStatus == entity.StatusDelivered || newStatus == entity.StatusOffline
	case entity.StatusDelivered:
		return newStatus == entity.StatusRead || newStatus == entity.StatusRecall
	case entity.StatusOffline:
		return newStatus == entity.StatusDelivered
	case entity.StatusRead:
		return newStatus == entity.StatusRecall
	default:
		return false
	}
}

// ToWSMessage 转换为WebSocket消息
func (m ChatMessage) ToWSMessage() WSMessage {
	return WSMessage{
		Code: 200,
		Msg:  "success",
		Data: m.Message,
	}
}

package dto

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

// messageSender 原生 WebSocket 消息发送器：直接发送 JSON 文本帧（无 socket.io 协议封装）
type messageSender struct{}

// NewMessageSender 创建原生消息发送器
func NewMessageSender() MessageSender {
	return messageSender{}
}

// Send 序列化并发送 JSON 消息
func (messageSender) Send(conn *websocket.Conn, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetWriteDeadline(time.Time{})
	return conn.WriteMessage(websocket.TextMessage, data)
}

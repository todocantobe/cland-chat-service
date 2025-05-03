package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type SocketIOProtocol struct {
}

func NewSocketIOProtocol() *SocketIOProtocol {
	return &SocketIOProtocol{}
}

// SocketIOMessage represents a Socket.IO protocol message
type SocketIOMessage struct {
	Type      int             `json:"type"` // Socket.IO packet type
	Namespace string          `json:"nsp"`  // Namespace
	ID        int             `json:"id"`   // Message ID (for ACK)
	Data      json.RawMessage `json:"data"` // Raw message data
}

const (
	PacketTypeConnect     = 0
	PacketTypeDisconnect  = 1
	PacketTypeEvent       = 2
	PacketTypeAck         = 3
	PacketTypeError       = 4
	PacketTypeBinaryEvent = 5
	PacketTypeBinaryAck   = 6
	PacketTypePing        = 8
	PacketTypePong        = 9
)

// SendConnect 发送 Socket.IO 连接确认
func (p *SocketIOProtocol) SendConnect(conn *websocket.Conn) error {
	ack := SocketIOMessage{
		Type:      PacketTypeConnect,
		Namespace: "/",
		Data:      json.RawMessage(`{"sid":"` + conn.RemoteAddr().String() + `","pingInterval":25000,"pingTimeout":5000}`),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// SendEvent 发送 Socket.IO 事件
func (p *SocketIOProtocol) SendEvent(conn *websocket.Conn, event string, data interface{}, namespace string) error {
	payload := []interface{}{event, data}
	msgData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := SocketIOMessage{
		Type:      PacketTypeEvent,
		Namespace: namespace,
		Data:      msgData,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, msgBytes)
}

// parseMessage 解析 Socket.IO 协议消息
func (p *SocketIOProtocol) parseMessage(data []byte) (*SocketIOMessage, error) {
	var msg SocketIOMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse Socket.IO message: %w", err)
	}
	return &msg, nil
}

// parseEventData 解析事件数据
func (p *SocketIOProtocol) parseEventData(msg *SocketIOMessage) (string, []interface{}, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(msg.Data, &raw); err != nil {
		return "", nil, err
	}

	if len(raw) == 0 {
		return "", nil, fmt.Errorf("empty event data")
	}

	var event string
	if err := json.Unmarshal(raw[0], &event); err != nil {
		return "", nil, err
	}

	args := make([]interface{}, len(raw)-1)
	for i, arg := range raw[1:] {
		var val interface{}
		if err := json.Unmarshal(arg, &val); err != nil {
			return "", nil, err
		}
		args[i] = val
	}

	return event, args, nil
}

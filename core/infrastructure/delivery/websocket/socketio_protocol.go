package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

// SocketIOProtocol implements Socket.IO 4.x protocol
type SocketIOProtocol struct {
	upgrader websocket.Upgrader
}

// NewSocketIOProtocol creates a new Socket.IO protocol handler
func NewSocketIOProtocol() *SocketIOProtocol {
	return &SocketIOProtocol{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// SocketIOMessage represents a Socket.IO protocol message
type SocketIOMessage struct {
	Type      int             `json:"type"` // Socket.IO packet type
	Namespace string          `json:"nsp"`  // Namespace
	ID        int             `json:"id"`   // Message ID (for ACK)
	Data      json.RawMessage `json:"data"` // Raw message data
}

const (
	// Socket.IO packet types
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

// parseMessage parses raw WebSocket message into Socket.IO protocol format
func (p *SocketIOProtocol) parseMessage(raw []byte) (*SocketIOMessage, error) {
	var msg SocketIOMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse Socket.IO message: %w", err)
	}
	return &msg, nil
}

// encodeMessage encodes Socket.IO message to raw WebSocket format
func (p *SocketIOProtocol) encodeMessage(msg *SocketIOMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Socket.IO message: %w", err)
	}
	return data, nil
}

// handleMessage processes incoming Socket.IO messages
func (p *SocketIOProtocol) handleMessage(conn *websocket.Conn, raw []byte) error {
	msg, err := p.parseMessage(raw)
	if err != nil {
		return err
	}

	switch msg.Type {
	case PacketTypeConnect:
		return p.handleConnect(conn, msg)
	case PacketTypeDisconnect:
		return p.handleDisconnect(conn, msg)
	case PacketTypeEvent:
		return p.handleEvent(conn, msg)
	case PacketTypePing:
		return p.handlePing(conn, msg)
	default:
		return fmt.Errorf("unsupported Socket.IO packet type: %d", msg.Type)
	}
}

// handleConnect handles connection handshake
func (p *SocketIOProtocol) handleConnect(conn *websocket.Conn, msg *SocketIOMessage) error {
	// Validate namespace
	if msg.Namespace != "/" {
		return fmt.Errorf("invalid namespace: %s", msg.Namespace)
	}

	// Send connection acknowledgement
	ack := SocketIOMessage{
		Type:      PacketTypeConnect,
		Namespace: msg.Namespace,
		Data:      json.RawMessage(`{"sid":"` + conn.RemoteAddr().String() + `"}`),
	}

	ackData, err := p.encodeMessage(&ack)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, ackData)
}

// handleDisconnect handles disconnection
func (p *SocketIOProtocol) handleDisconnect(conn *websocket.Conn, msg *SocketIOMessage) error {
	return conn.Close()
}

// handleEvent handles incoming events
func (p *SocketIOProtocol) handleEvent(conn *websocket.Conn, msg *SocketIOMessage) error {
	// TODO: Implement event handling
	return nil
}

// handlePing handles ping/pong heartbeat
func (p *SocketIOProtocol) handlePing(conn *websocket.Conn, msg *SocketIOMessage) error {
	pong := SocketIOMessage{
		Type: PacketTypePong,
	}
	pongData, err := p.encodeMessage(&pong)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, pongData)
}

// SendEvent sends an event to the client
func (p *SocketIOProtocol) SendEvent(conn *websocket.Conn, event string, data interface{}, namespace string) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := SocketIOMessage{
		Type:      PacketTypeEvent,
		Namespace: namespace,
		Data:      payload,
	}

	msgData, err := p.encodeMessage(&msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, msgData)
}

// parseEventData parses event data from Socket.IO message
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

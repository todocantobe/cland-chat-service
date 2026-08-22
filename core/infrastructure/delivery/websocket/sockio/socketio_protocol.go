package sockio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// EngineIOProtocol implements Engine.IO v4 protocol
type EngineIOProtocol struct {
}

func NewEngineIOProtocol() *EngineIOProtocol {
	return &EngineIOProtocol{}
}

// Packet types for Engine.IO v4
const (
	PacketTypeOpen    = "0"
	PacketTypeClose   = "1"
	PacketTypePing    = "2"
	PacketTypePong    = "3"
	PacketTypeMessage = "4"
	PacketTypeUpgrade = "5"
	PacketTypeNoop    = "6"
)

// Socket.IO protocol types (sent as Engine.IO message type "4")
const (
	SocketIOPacketConnect      = "0"
	SocketIOPacketDisconnect   = "1"
	SocketIOPacketEvent        = "2"
	SocketIOPacketAck          = "3"
	SocketIOPacketConnectError = "4"
	SocketIOPacketBinaryEvent  = "5"
	SocketIOPacketBinaryAck    = "6"
)

// HandshakeData represents the handshake response data
type HandshakeData struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
	MaxPayload   int      `json:"maxPayload"`
}

// SendHandshake sends the Engine.IO v4 handshake response for polling transport
func (p *EngineIOProtocol) SendHandshake(w http.ResponseWriter, sid string) error {
	data := HandshakeData{
		SID:          sid,
		Upgrades:     []string{"websocket"},
		PingInterval: 25000,   // 25 seconds in milliseconds
		PingTimeout:  20000,   // 20 seconds in milliseconds
		MaxPayload:   1000000, // 1MB
	}

	// Set required headers
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS support

	// Engine.IO packet format:
	// 0{"sid":"...","upgrades":[...],...}
	// 注意：必须用 json.Marshal 而非 json.Encoder（后者会附加换行符，违反 Engine.IO 载荷规范）
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode handshake data: %w", err)
	}

	response := append([]byte{'0'}, jsonData...) // Packet type '0' (open)
	_, err = w.Write(response)
	return err
}

// BuildSocketIOPacket constructs a Socket.IO v4 protocol message.
// v4 规范：<type>[<namespace>,]<data>，命名空间仅在非默认("/")时出现。
// 默认命名空间的事件包形如 2["event",data]，连接确认形如 0{"sid":"..."}。
func (p *EngineIOProtocol) BuildSocketIOPacket(packetType string, namespace string, data interface{}) (string, error) {
	var builder strings.Builder
	builder.WriteString(packetType) // Socket.IO packet type

	// v4：仅当命名空间非默认时才写 <namespace>, 前缀
	if namespace != "" && namespace != "/" {
		builder.WriteString(namespace)
		builder.WriteString(",")
	}

	switch v := data.(type) {
	case string:
		builder.WriteString(v)
	case []byte:
		builder.Write(v)
	case nil:
		// 无载荷（如纯 connect 确认 "40"、disconnect "41"）
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		builder.Write(jsonData)
	}

	return builder.String(), nil
}

// ParseSocketIOPacket parses a Socket.IO v4 protocol message.
// v4 规范：<type>[<namespace>,][<ackId>]<payload>，命名空间仅在剩余部分以 '/' 开头时出现。
// 例如：
//
//	"0"                     -> CONNECT, ns=/, 无载荷（len==1 合法）
//	"0{\"token\":\"t\"}"       -> CONNECT, ns=/, 带认证载荷
//	"0/admin"               -> CONNECT, ns=/admin
//	"0/admin,{\"token\":\"t\"}" -> CONNECT, ns=/admin, 带载荷
//	"1"                     -> DISCONNECT
//	"2[\"message\",{...}]"  -> EVENT, ns=/
//	"2/admin,[\"message\",...]" -> EVENT, ns=/admin
//	"3<ackId>[,payload]"    -> ACK
func (p *EngineIOProtocol) ParseSocketIOPacket(data []byte) (packetType string, namespace string, payload []byte, ackID int, err error) {
	if len(data) == 0 {
		return "", "", nil, 0, fmt.Errorf("invalid Socket.IO packet length")
	}

	packetType = string(data[0])
	remaining := data[1:]

	// 命名空间：仅当以 '/' 开头时存在（v4 规范，不再用逗号探测——
	// 旧实现把事件 JSON 数组内的逗号误判为命名空间分隔符）
	namespace = "/"
	if len(remaining) > 0 && remaining[0] == '/' {
		nsEnd := bytes.IndexByte(remaining, ',')
		if nsEnd == -1 {
			namespace = string(remaining)
			remaining = nil
		} else {
			namespace = string(remaining[:nsEnd])
			remaining = remaining[nsEnd+1:]
		}
	}

	// 可选的 ACK ID（数字前缀）
	ackEnd := 0
	for ackEnd < len(remaining) && remaining[ackEnd] >= '0' && remaining[ackEnd] <= '9' {
		ackID = ackID*10 + int(remaining[ackEnd]-'0')
		ackEnd++
	}
	if ackEnd > 0 {
		remaining = remaining[ackEnd:]
	}

	// 兼容垫片：剥离类型/ACK ID 后的单个前导逗号。
	// 标准 v4 包不会出现（命名空间已在上方消费），但旧自定义格式
	// （"2,[\"message\",...]"）与 ACK 带数据（"3<id>,<data>"）会带逗号。
	if len(remaining) > 0 && remaining[0] == ',' {
		remaining = remaining[1:]
	}

	payload = remaining
	return packetType, namespace, payload, ackID, nil
}

// SendPacket sends an Engine.IO packet over WebSocket
func (p *EngineIOProtocol) SendPacket(conn *websocket.Conn, packetType string, data interface{}) error {
	var msg string
	switch v := data.(type) {
	case string:
		msg = packetType + v
	case []byte:
		msg = packetType + string(v)
	case nil:
		msg = packetType // 无载荷（如 ping "2"、pong "3"）
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		msg = packetType + string(jsonData)
	}
	return conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// SendPollingPackets sends multiple packets in polling format
func (p *EngineIOProtocol) SendPollingPackets(w http.ResponseWriter, packets []string) error {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	_, err := w.Write([]byte(strings.Join(packets, "\x1e")))
	return err
}

// ParsePacket parses an incoming Engine.IO packet
func (p *EngineIOProtocol) ParsePacket(data []byte) (packetType string, payload []byte, err error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("empty packet")
	}
	packetType = string(data[0])
	if len(data) > 1 {
		payload = data[1:]
	}
	return packetType, payload, nil
}

// ParseEventPayload parses a Socket.IO event payload in the format ["eventName", eventData]
// 事件数据可缺省（如 ["join"]），此时 eventData 返回 nil。
func (p *EngineIOProtocol) ParseEventPayload(payload []byte) (eventName string, eventData []byte, err error) {
	if len(payload) == 0 {
		return "", nil, fmt.Errorf("empty payload")
	}

	// Parse the JSON array
	var arr []json.RawMessage
	if err := json.Unmarshal(payload, &arr); err != nil {
		return "", nil, fmt.Errorf("invalid event payload format: %w", err)
	}

	if len(arr) < 1 {
		return "", nil, fmt.Errorf("event payload must contain at least 1 element")
	}

	// Extract event name
	if err := json.Unmarshal(arr[0], &eventName); err != nil {
		return "", nil, fmt.Errorf("failed to parse event name: %w", err)
	}

	// Return event data as raw JSON (may be absent)
	if len(arr) > 1 {
		eventData = arr[1]
	}
	return eventName, eventData, nil
}

// HandlePing starts the heartbeat mechanism
func (p *EngineIOProtocol) HandlePing(conn *websocket.Conn, interval time.Duration, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := p.SendPacket(conn, PacketTypePing, nil); err != nil {
			return
		}

		// Wait for pong with timeout
		conn.SetReadDeadline(time.Now().Add(timeout))
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.SetReadDeadline(time.Time{}) // Reset deadline
	}
}

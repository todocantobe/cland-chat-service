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

// Socket.IO v4 扩展常量（补充原有缺失的类型）
const (
	// 多位数字类型标识（Socket.IO v4 标准）
	SocketIOPacketEventV4       = "42" // 文本事件包（最常用，对应客户端 emit）
	SocketIOPacketBinaryEventV4 = "45" // 二进制事件包
	SocketIOPacketAckV4         = "43" // 文本 ACK 包
	SocketIOPacketBinaryAckV4   = "46" // 二进制 ACK 包

	// 命名空间分隔符（明确常量，避免硬编码）
	namespaceSeparator = ","
	namespaceDefault   = "/"
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
	response := new(bytes.Buffer)
	response.WriteByte('0') // Packet type '0' (open)

	encoder := json.NewEncoder(response)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode handshake data: %w", err)
	}

	// json.Encoder adds a newline, but Engine.IO spec doesn't require it
	// We'll keep it for compatibility with most clients

	// Write the complete response
	_, err := w.Write(response.Bytes())
	return err
}

// BuildSocketIOPacket constructs a Socket.IO protocol message according to v4 spec
func (p *EngineIOProtocol) BuildSocketIOPacket(packetType string, namespace string, data interface{}) (string, error) {
	var builder strings.Builder
	builder.WriteString(packetType) // Socket.IO packet type

	// Add binary attachments count (0 for now, as we don't support binary)
	// Format: <packet type>[<# of binary attachments>-][<namespace>,][<acknowledgment id>][JSON-stringified payload]
	
	// Handle namespace according to spec
	if namespace != "" && namespace != "/" {
		builder.WriteString(namespace)
		builder.WriteString(",")
	} else if namespace == "/" {
		// Default namespace, no namespace in packet
	} else {
		// No namespace specified, use default
		// No namespace in packet for default
	}

	// Handle data
	switch v := data.(type) {
	case string:
		builder.WriteString(v)
	case []byte:
		builder.Write(v)
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		builder.Write(jsonData)
	}

	return builder.String(), nil
}

// ParseSocketIOPacket 解析 Socket.IO v4 数据包（无魔法数，严格使用常量）
func (p *EngineIOProtocol) ParseSocketIOPacket(data []byte) (packetType string, namespace string, payload []byte, ackID int, err error) {
	if len(data) == 0 {
		return "", "", nil, 0, fmt.Errorf("invalid Socket.IO packet length: empty data")
	}

	// 提取 PacketType（支持 1-2 位数字，Socket.IO v4 标准）
	packetTypeEnd := 0
	for packetTypeEnd < len(data) && data[packetTypeEnd] >= '0' && data[packetTypeEnd] <= '9' {
		packetTypeEnd++
	}
	if packetTypeEnd == 0 {
		return "", "", nil, 0, fmt.Errorf("invalid packet type: no numeric prefix")
	}
	packetType = string(data[:packetTypeEnd])
	remaining := data[packetTypeEnd:]

	// 解析命名空间（默认值：namespaceDefault）
	namespace = namespaceDefault
	if len(remaining) > 0 {
		// 场景1：显式指定命名空间（以 "/" 开头）
		if remaining[0] == '/' {
			nsEnd := bytes.IndexByte(remaining, namespaceSeparator[0])
			if nsEnd == -1 {
				// 无分隔符，剩余部分全为命名空间
				namespace = string(remaining)
				remaining = nil
			} else {
				namespace = string(remaining[:nsEnd])
				remaining = remaining[nsEnd+1:] // 跳过分隔符
			}
		} else if remaining[0] == namespaceSeparator[0] {
			// 场景2：无命名空间，直接以分隔符开头
			remaining = remaining[1:]
		}
		// 场景3：无命名空间且无分隔符，直接使用默认值
	}

	// 处理 ACK 包（仅针对 ACK 类型的数据包）
	ackTypes := map[string]bool{
		SocketIOPacketAck:         true,
		SocketIOPacketBinaryAck:   true,
		SocketIOPacketAckV4:       true,
		SocketIOPacketBinaryAckV4: true,
	}
	if ackTypes[packetType] {
		ackEnd := 0
		for ackEnd < len(remaining) && remaining[ackEnd] >= '0' && remaining[ackEnd] <= '9' {
			ackID = ackID*10 + int(remaining[ackEnd]-'0')
			ackEnd++
		}
		if ackEnd > 0 {
			remaining = remaining[ackEnd:]
			// 跳过 ACK ID 后的分隔符
			if len(remaining) > 0 && remaining[0] == namespaceSeparator[0] {
				remaining = remaining[1:]
			}
		}
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
func (p *EngineIOProtocol) ParseEventPayload(payload []byte) (eventName string, eventData []byte, err error) {
	if len(payload) == 0 {
		return "", nil, fmt.Errorf("empty payload")
	}

	// Parse the JSON array
	var arr []json.RawMessage
	if err := json.Unmarshal(payload, &arr); err != nil {
		return "", nil, fmt.Errorf("invalid event payload format: %w", err)
	}

	if len(arr) < 2 {
		return "", nil, fmt.Errorf("event payload must contain at least 2 elements")
	}

	// Extract event name
	if err := json.Unmarshal(arr[0], &eventName); err != nil {
		return "", nil, fmt.Errorf("failed to parse event name: %w", err)
	}

	// Return event data as raw JSON
	eventData = arr[1]
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

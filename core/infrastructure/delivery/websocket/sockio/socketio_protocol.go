package sockio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// BuildSocketIOPacket constructs a Socket.IO protocol message
func (p *EngineIOProtocol) BuildSocketIOPacket(packetType string, namespace string, data interface{}) (string, error) {
	var builder strings.Builder
	builder.WriteString(packetType) // Socket.IO packet type

	if namespace != "" && namespace != "/" {
		builder.WriteString(namespace)
	}
	builder.WriteString(",")
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

// ParseSocketIOPacket parses a Socket.IO protocol message according to the v4 protocol
func (p *EngineIOProtocol) ParseSocketIOPacket(data []byte) (packetType string, namespace string, payload []byte, ackID int, err error) {
	if len(data) < 2 {
		return "", "", nil, 0, fmt.Errorf("invalid Socket.IO packet length")
	}

	packetType = string(data[0])
	remaining := data[1:]

	// Parse namespace (optional)
	namespace = "/"
	if len(remaining) > 0 {
		nsEnd := bytes.IndexByte(remaining, ',')
		if nsEnd == -1 {
			// No comma found, entire remaining is namespace
			namespace = string(remaining)
			remaining = nil
		} else {
			namespace = string(remaining[:nsEnd])
			remaining = remaining[nsEnd+1:]
		}

		// Normalize empty namespace to "/"
		if namespace == "" {
			namespace = "/"
		}
	}

	// Handle ACK packets (type 3 or 6)
	if packetType == SocketIOPacketAck || packetType == SocketIOPacketBinaryAck {
		// Extract ACK ID (numeric prefix before payload)
		ackEnd := 0
		for ackEnd < len(remaining) && remaining[ackEnd] >= '0' && remaining[ackEnd] <= '9' {
			ackID = ackID*10 + int(remaining[ackEnd]-'0')
			ackEnd++
		}
		if ackEnd > 0 {
			remaining = remaining[ackEnd:]
			if len(remaining) > 0 && remaining[0] == ',' {
				remaining = remaining[1:]
			}
		}
	}

	// Handle EVENT/BINARY_EVENT packets (type 2 or 5)
	if packetType == SocketIOPacketEvent || packetType == SocketIOPacketBinaryEvent {
		// Check for JSON array format (e.g. ["event", data] or ["event", data, ackId])
		if len(remaining) > 0 && remaining[0] == '[' {
			end := len(remaining) - 1
			if remaining[end] == ']' {
				// Extract event data (may contain ACK ID)
				lastComma := bytes.LastIndexByte(remaining, ',')
				if lastComma != -1 {
					// Check if last element is ACK ID (number)
					ackStr := string(remaining[lastComma+1 : end])
					if ackNum, err := strconv.Atoi(ackStr); err == nil {
						ackID = ackNum
						remaining = remaining[:lastComma]
						remaining = append(remaining, ']')
					}
				}
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

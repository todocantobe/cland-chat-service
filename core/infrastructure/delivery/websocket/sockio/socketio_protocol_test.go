package sockio

import (
	"testing"
)

// TestParseSocketIOPacket_StandardPackets 覆盖 ISSUE BAS-ISSUE-002 证据中的标准包解析
func TestParseSocketIOPacket_StandardPackets(t *testing.T) {
	p := NewEngineIOProtocol()

	cases := []struct {
		name          string
		data          string
		wantType      string
		wantNamespace string
		wantPayload   string
		wantAckID     int
	}{
		// 证据①：标准 connect 包 "40"（Engine.IO 4 + Socket.IO 0），旧实现 len<2 拒绝
		{"connect default ns", "0", "0", "/", "", 0},
		// 证据③：标准事件包 42["message",{...}]，旧实现把 JSON 内逗号误判为命名空间分隔符
		{"event default ns", `2["message",{"a":1}]`, "2", "/", `["message",{"a":1}]`, 0},
		{"event no data", `2["join"]`, "2", "/", `["join"]`, 0},
		{"event ns", `2/admin,["message",{"a":1}]`, "2", "/admin", `["message",{"a":1}]`, 0},
		{"event ns no data", `2/admin,["join"]`, "2", "/admin", `["join"]`, 0},
		{"connect ns", "0/admin", "0", "/admin", "", 0},
		{"connect ns with payload", `0/admin,{"token":"t"}`, "0", "/admin", `{"token":"t"}`, 0},
		{"connect with payload", `0{"token":"t"}`, "0", "/", `{"token":"t"}`, 0},
		{"disconnect", "1", "1", "/", "", 0},
		{"disconnect ns", "1/admin", "1", "/admin", "", 0},
		{"ack", "3", "3", "/", "", 0},
		{"ack with id", "312", "3", "/", "", 12},
		{"ack with id and payload", `312,["done"]`, "3", "/", `["done"]`, 12},
		{"ack ns with id", "3/admin,12", "3", "/admin", "", 12},
		// 自定义格式（旧版客户端曾用）也应兼容：42,["message",{...}]
		{"legacy custom comma format", `2,["message",{"a":1}]`, "2", "/", `["message",{"a":1}]`, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packetType, namespace, payload, ackID, err := p.ParseSocketIOPacket([]byte(tc.data))
			if err != nil {
				t.Fatalf("ParseSocketIOPacket(%q) unexpected error: %v", tc.data, err)
			}
			if packetType != tc.wantType {
				t.Errorf("packetType = %q, want %q", packetType, tc.wantType)
			}
			if namespace != tc.wantNamespace {
				t.Errorf("namespace = %q, want %q", namespace, tc.wantNamespace)
			}
			if string(payload) != tc.wantPayload {
				t.Errorf("payload = %q, want %q", payload, tc.wantPayload)
			}
			if ackID != tc.wantAckID {
				t.Errorf("ackID = %d, want %d", ackID, tc.wantAckID)
			}
		})
	}
}

func TestParseSocketIOPacket_Empty(t *testing.T) {
	p := NewEngineIOProtocol()
	if _, _, _, _, err := p.ParseSocketIOPacket(nil); err == nil {
		t.Fatal("expected error for empty packet")
	}
}

// TestBuildSocketIOPacket_StandardFormat 修复前 BuildSocketIOPacket 会产出 2,["event",...] 非标格式
func TestBuildSocketIOPacket_StandardFormat(t *testing.T) {
	p := NewEngineIOProtocol()

	// 默认命名空间事件包：标准 2["message",{...}]，无多余逗号
	got, err := p.BuildSocketIOPacket(SocketIOPacketEvent, "/", []interface{}{"message", map[string]string{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	want := `2["message",{"k":"v"}]`
	if got != want {
		t.Errorf("BuildSocketIOPacket event = %q, want %q", got, want)
	}

	// 默认命名空间 connect 确认：0{"sid":"..."}
	got, err = p.BuildSocketIOPacket(SocketIOPacketConnect, "/", map[string]string{"sid": "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `0{"sid":"s1"}` {
		t.Errorf("BuildSocketIOPacket connect = %q, want %q", got, `0{"sid":"s1"}`)
	}

	// 非默认命名空间：2/admin,["message",...]
	got, err = p.BuildSocketIOPacket(SocketIOPacketEvent, "/admin", []interface{}{"message", "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `2/admin,["message","hi"]` {
		t.Errorf("BuildSocketIOPacket ns event = %q, want %q", got, `2/admin,["message","hi"]`)
	}

	// 无载荷（nil）
	got, err = p.BuildSocketIOPacket(SocketIOPacketConnect, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0" {
		t.Errorf("BuildSocketIOPacket nil = %q, want %q", got, "0")
	}
}

// TestParseEventPayload 事件载荷解析：标准 2 元素与缺省数据 1 元素
func TestParseEventPayload(t *testing.T) {
	p := NewEngineIOProtocol()

	eventName, eventData, err := p.ParseEventPayload([]byte(`["message",{"dst":"U:user_002"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if eventName != "message" {
		t.Errorf("eventName = %q, want message", eventName)
	}
	if string(eventData) != `{"dst":"U:user_002"}` {
		t.Errorf("eventData = %q", eventData)
	}

	// 1 元素事件（无数据）：["join"]
	eventName, eventData, err = p.ParseEventPayload([]byte(`["join"]`))
	if err != nil {
		t.Fatal(err)
	}
	if eventName != "join" {
		t.Errorf("eventName = %q, want join", eventName)
	}
	if eventData != nil {
		t.Errorf("eventData = %q, want nil", eventData)
	}
}

// TestSendHandshake_NoTrailingNewline 握手响应不允许有 json.Encoder 的换行符
func TestSendHandshake_NoTrailingNewline(t *testing.T) {
	// 通过 BuildSocketIOPacket 验证内部 JSON 序列化行为：json.Encoder 会加 \n，json.Marshal 不会
	p := NewEngineIOProtocol()
	packet, err := p.BuildSocketIOPacket(SocketIOPacketConnect, "/", map[string]string{"sid": "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(packet) > 0 && packet[len(packet)-1] == '\n' {
		t.Errorf("packet ends with newline: %q", packet)
	}
}

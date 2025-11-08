package sockio

import (
	"fmt"
	"time"

	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/handler"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ProtocolHandler 协议层处理器，专门处理协议相关的逻辑
type ProtocolHandler struct {
	logger      *zap.Logger
	protocol    *EngineIOProtocol
	wsHandler   *handler.Handler
	connManager interface{}
}

// NewProtocolHandler 创建协议处理器
func NewProtocolHandler(logger *zap.Logger, protocol *EngineIOProtocol, wsHandler *handler.Handler) *ProtocolHandler {
	return &ProtocolHandler{
		logger:    logger,
		protocol:  protocol,
		wsHandler: wsHandler,
	}
}

// HandleEngineIOPacket 处理 Engine.IO 数据包
func (p *ProtocolHandler) HandleEngineIOPacket(conn *websocket.Conn, packetType string, payload []byte, clandCID string) error {
	switch packetType {
	case PacketTypePing:
		// 处理 Ping 包
		return p.handlePing(conn, clandCID)

	case PacketTypePong:
		// 处理 Pong 包
		p.logger.Debug("Received pong from client", zap.String("clandCID", clandCID))
		return nil

	case PacketTypeMessage:
		// 处理消息包
		return p.handleSocketIOPacket(conn, payload, clandCID)

	case PacketTypeClose:
		// 处理关闭包
		p.logger.Info("Received close packet from client", zap.String("clandCID", clandCID))
		return fmt.Errorf("client requested close")

	default:
		p.logger.Warn("Unsupported Engine.IO packet type", 
			zap.String("packetType", packetType),
			zap.String("clandCID", clandCID))
		return nil
	}
}

// handlePing 处理 Ping 包
func (p *ProtocolHandler) handlePing(conn *websocket.Conn, clandCID string) error {
	// 发送 Pong 响应
	_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := p.protocol.SendPacket(conn, PacketTypePong, "probe"); err != nil {
		p.logger.Error("Failed to send pong", 
			zap.Error(err),
			zap.String("clandCID", clandCID))
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

// handleSocketIOPacket 处理 Socket.IO 数据包
func (p *ProtocolHandler) handleSocketIOPacket(conn *websocket.Conn, payload []byte, clandCID string) error {
	log := p.logger.With(zap.String("clandCID", clandCID))

	// 解析 Socket.IO 数据包
	sioType, namespace, sioPayload, ackID, err := p.protocol.ParseSocketIOPacket(payload)
	if err != nil {
		log.Error("Failed to parse Socket.IO packet", zap.Error(err))
		return err
	}

	switch {
	// 处理连接包
	case sioType == SocketIOPacketConnect:
		return p.handleConnect(conn, namespace, clandCID)

	// 处理事件包
	case p.isEventPacket(sioType):
		return p.handleEvent(conn, sioType, namespace, sioPayload, ackID, clandCID)

	// 处理断开连接包
	case sioType == SocketIOPacketDisconnect:
		p.logger.Info("Client disconnected from namespace", 
			zap.String("namespace", namespace),
			zap.String("clandCID", clandCID))
		return fmt.Errorf("client disconnected")

	// 处理错误包
	case sioType == SocketIOPacketConnectError:
		p.logger.Warn("Received connect error from client", 
			zap.String("namespace", namespace),
			zap.String("clandCID", clandCID))
		return fmt.Errorf("client reported connect error")

	// 未知类型
	default:
		log.Warn("Unsupported Socket.IO packet type", 
			zap.String("sioType", sioType),
			zap.String("namespace", namespace))
		return nil
	}
}

// handleConnect 处理连接包
func (p *ProtocolHandler) handleConnect(conn *websocket.Conn, namespace string, clandCID string) error {
	p.logger.Info("Client connected to namespace", 
		zap.String("namespace", namespace),
		zap.String("clandCID", clandCID))

	// 发送连接确认
	ackData := map[string]string{"sid": clandCID}
	ackPacket, err := p.protocol.BuildSocketIOPacket(SocketIOPacketConnect, namespace, ackData)
	if err != nil {
		p.logger.Error("Failed to build connect ack", zap.Error(err))
		return err
	}

	_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := p.protocol.SendPacket(conn, PacketTypeMessage, ackPacket); err != nil {
		p.logger.Error("Failed to send connect ack", zap.Error(err))
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

// handleEvent 处理事件包
func (p *ProtocolHandler) handleEvent(conn *websocket.Conn, sioType string, namespace string, sioPayload []byte, ackID int, clandCID string) error {
	log := p.logger.With(zap.String("clandCID", clandCID))

	// 解析事件数据
	eventName, eventData, err := p.protocol.ParseEventPayload(sioPayload)
	if err != nil {
		log.Error("Failed to parse event payload", zap.Error(err))
		return err
	}

	// 处理 ACK 响应
	if ackID > 0 {
		return p.handleEventWithAck(conn, sioType, namespace, eventName, eventData, ackID, clandCID)
	} else {
		return p.handleEventWithoutAck(conn, namespace, eventName, eventData, clandCID)
	}
}

// handleEventWithAck 处理需要 ACK 的事件
func (p *ProtocolHandler) handleEventWithAck(conn *websocket.Conn, sioType string, namespace string, eventName string, eventData []byte, ackID int, clandCID string) error {
	log := p.logger.With(zap.String("clandCID", clandCID))

	// 调用业务处理器处理消息
	response, err := p.wsHandler.HandleMessageWithAck(conn, eventName, string(eventData))
	if err != nil {
		log.Error("Failed to handle message with ack", 
			zap.String("eventName", eventName), 
			zap.Error(err))
		return err
	}

	// 构建 ACK 包 - 根据标准协议，ACK 包格式为 [responseData]
	var ackPacketType string
	switch sioType {
	case SocketIOPacketBinaryEvent, SocketIOPacketBinaryEventV4:
		ackPacketType = SocketIOPacketBinaryAckV4
	default:
		ackPacketType = SocketIOPacketAckV4
	}

	// 根据标准协议，ACK 包的数据格式是 [responseData]
	// ACK ID 已经在包的开头编码了
	ackPacket, err := p.protocol.BuildSocketIOPacket(ackPacketType, namespace, []interface{}{response})
	if err != nil {
		log.Error("Failed to build ack packet", zap.Error(err))
		return err
	}

	// 发送 ACK 响应
	_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := p.protocol.SendPacket(conn, PacketTypeMessage, ackPacket); err != nil {
		log.Error("Failed to send ack response", zap.Error(err))
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

// handleEventWithoutAck 处理不需要 ACK 的事件
func (p *ProtocolHandler) handleEventWithoutAck(conn *websocket.Conn, namespace string, eventName string, eventData []byte, clandCID string) error {
	p.wsHandler.HandleEvent(conn, namespace, eventName, string(eventData))
	return nil
}

// isEventPacket 判断是否为事件包类型
func (p *ProtocolHandler) isEventPacket(sioType string) bool {
	eventPacketTypes := map[string]bool{
		SocketIOPacketEvent:        true,
		SocketIOPacketBinaryEvent:  true,
		SocketIOPacketEventV4:      true,
		SocketIOPacketBinaryEventV4: true,
	}
	return eventPacketTypes[sioType]
}

// SendEvent 发送事件（协议层方法）
func (p *ProtocolHandler) SendEvent(conn *websocket.Conn, namespace string, eventName string, data interface{}) error {
	packet, err := p.protocol.BuildSocketIOPacket(SocketIOPacketEventV4, namespace, []interface{}{eventName, data})
	if err != nil {
		return err
	}
	return p.protocol.SendPacket(conn, PacketTypeMessage, packet)
}

// SendError 发送错误（协议层方法）
func (p *ProtocolHandler) SendError(conn *websocket.Conn, namespace string, err error) error {
	errorData := map[string]interface{}{
		"error":   true,
		"message": err.Error(),
	}
	return p.SendEvent(conn, namespace, "error", errorData)
}

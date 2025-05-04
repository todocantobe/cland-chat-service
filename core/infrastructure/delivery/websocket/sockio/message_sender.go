package sockio

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
)

// SocketIOMessageSender implements dto.MessageSender interface
// using Socket.IO protocol for message delivery
var _ dto.MessageSender = (*SocketIOMessageSender)(nil)

// SocketIOMessageSender implements dto.MessageSender using Socket.IO protocol
type SocketIOMessageSender struct {
	protocol *EngineIOProtocol
	logger   *zap.Logger
}

func NewSocketIOMessageSender(protocol *EngineIOProtocol, logger *zap.Logger) *SocketIOMessageSender {
	return &SocketIOMessageSender{
		protocol: protocol,
		logger:   logger,
	}
}

func (s *SocketIOMessageSender) SendEvent(conn *websocket.Conn, namespace string, eventName string, data interface{}) error {
	packet, err := s.protocol.BuildSocketIOPacket(SocketIOPacketEvent, namespace, []interface{}{eventName, data})
	if err != nil {
		s.logger.Error("Failed to build event packet", zap.Error(err))
		return err
	}
	return s.protocol.SendPacket(conn, PacketTypeMessage, packet)
}

func (s *SocketIOMessageSender) SendError(conn *websocket.Conn, namespace string, err error) error {
	packet, err := s.protocol.BuildSocketIOPacket(SocketIOPacketEvent, namespace, map[string]string{
		"message": err.Error(),
	})
	if err != nil {
		s.logger.Error("Failed to build error packet", zap.Error(err))
		return err
	}
	return s.protocol.SendPacket(conn, PacketTypeMessage, packet)
}

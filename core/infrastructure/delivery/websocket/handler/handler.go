package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	cland_errors "cland.org/cland-chat-service/common/errors"
	"cland.org/cland-chat-service/core/application"
	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
	"github.com/gorilla/websocket"
)

type Handler struct {
	ChatService       *application.ChatService
	ConnectionManager *connection.Manager
	MessageSender     dto.MessageSender
}

// HandleMessage 处理客户端发来的 JSON 消息
func (h *Handler) HandleMessage(conn *websocket.Conn, data string) {
	var msg entity.Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		h.sendError(conn, "invalid message format")
		return
	}

	if err := h.processMessage(conn, msg); err != nil {
		h.sendError(conn, err.Error())
	}
}

// HandleError 记录连接错误
func (h *Handler) HandleError(conn *websocket.Conn, err error) {
	log.Println("socket error:", err)
}

// HandleDisconnect 记录断开
func (h *Handler) HandleDisconnect(conn *websocket.Conn, reason string) {
	log.Println("disconnected:", conn.RemoteAddr(), reason)
}

// processMessage 处理消息业务逻辑
func (h *Handler) processMessage(conn *websocket.Conn, msg entity.Message) error {
	ctx := context.Background()

	switch msg.MsgType {
	case entity.MsgTypeMessage, entity.MsgTypeNotification:
		if err := h.ChatService.SendMessage(ctx, &msg); err != nil {
			return err
		}
		return h.pushMessage(msg)
	case entity.MsgTypeAck:
		return h.ChatService.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusRead)
	default:
		return errors.New("unsupported message type")
	}
}

// pushMessage 推送消息给接收方
func (h *Handler) pushMessage(msg entity.Message) error {
	// 房间消息（前缀 room:）广播到房间
	if len(msg.Dst) > 5 && msg.Dst[:5] == "room:" {
		roomID := msg.Dst[5:]
		msg.Status = entity.StatusDelivered
		wsMsg := dto.FromEntity(msg).ToWSMessage()
		return h.ConnectionManager.BroadcastToRoom(wsMsg, roomID)
	}

	// 直发消息：解析接收方 ID（格式 "U:user_xxx" / "A:agent_xxx"）
	recipientID := msg.Dst
	if len(msg.Dst) > 2 && msg.Dst[1] == ':' {
		recipientID = msg.Dst[2:]
	}

	msg.Status = entity.StatusDelivered
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	if conn, ok := h.ConnectionManager.GetConnection(recipientID); ok {
		return h.MessageSender.Send(conn, wsMsg)
	}

	// 接收方离线，更新为离线状态
	return h.ChatService.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
}

// sendError 发送错误消息（统一错误消息体）
func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	_ = h.MessageSender.Send(conn, cland_errors.Err500)
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	for _, userID := range userIDs {
		if conn, ok := h.ConnectionManager.GetConnection(userID); ok {
			if err := h.MessageSender.Send(conn, wsMsg); err != nil {
				return err
			}
		}
	}
	return nil
}

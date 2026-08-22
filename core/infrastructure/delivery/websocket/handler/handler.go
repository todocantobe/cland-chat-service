package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gorilla/websocket"
)

// Handler 消息路由处理器。
// 连接注册唯一来源是 ConnectionManager（sockio 层建立连接时注册），
// 不再维护第二套 connections 映射（旧实现从未填充，导致定向转发永远走离线分支）。
type Handler struct {
	ChatUseCase       *usecase.ChatUseCase
	ConnectionManager *connection.Manager
	MessageSender     dto.MessageSender
}

func (h *Handler) HandleMessage(conn *websocket.Conn, data string) {
	// Parse message
	var msg entity.Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		h.sendError(conn, "invalid message format")
		return
	}

	// Process message
	if err := h.processMessage(conn, msg); err != nil {
		h.sendError(conn, err.Error())
	}
}

func (h *Handler) HandleError(conn *websocket.Conn, err error) {
	log.Println("socket error:", err)
}

// HandleDisconnect 连接断开时清理注册（含房间成员）
func (h *Handler) HandleDisconnect(conn *websocket.Conn, userID string, reason string) {
	log.Println("disconnected:", conn.RemoteAddr(), reason)
	h.ConnectionManager.RemoveConnection(userID)
}

// processMessage 处理消息业务逻辑
func (h *Handler) processMessage(conn *websocket.Conn, msg entity.Message) error {
	ctx := context.Background()

	switch msg.MsgType {
	case entity.MsgTypeMessage, entity.MsgTypeNotification:
		if err := h.ChatUseCase.SendMessage(ctx, &msg); err != nil {
			return err
		}
		return h.pushMessage(msg)
	case entity.MsgTypeAck:
		return h.ChatUseCase.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusRead)
	default:
		return errors.New("unsupported message type")
	}
}

// pushMessage 推送消息给接收方
func (h *Handler) pushMessage(msg entity.Message) error {
	// Handle room messages (prefix with "room:")
	if strings.HasPrefix(msg.Dst, "room:") {
		roomID := msg.Dst[len("room:"):]
		msg.Status = entity.StatusDelivered
		wsMsg := dto.FromEntity(msg).ToWSMessage()
		return h.broadcastToRoom(wsMsg, roomID)
	}

	// Handle direct messages
	recipientID := msg.Dst
	if len(msg.Dst) > 2 && msg.Dst[1] == ':' {
		recipientID = msg.Dst[2:]
	}

	msg.Status = entity.StatusDelivered
	wsMsg := dto.FromEntity(msg).ToWSMessage()

	// 从 ConnectionManager 单一注册表查接收方
	if conn, ok := h.ConnectionManager.GetConnection(recipientID); ok {
		return h.MessageSender.SendEvent(conn, "/", "message", wsMsg)
	}

	// 接收方离线，更新为离线状态
	return h.ChatUseCase.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
}

// broadcastToRoom 按 Socket.IO 帧协议广播到房间内所有在线成员
func (h *Handler) broadcastToRoom(wsMsg dto.WSMessage, roomID string) error {
	conns := h.ConnectionManager.GetRoomConnections(roomID)
	for _, conn := range conns {
		if err := h.MessageSender.SendEvent(conn, "/", "message", wsMsg); err != nil {
			return err
		}
	}
	return nil
}

// sendError 发送错误消息（标准事件帧：2["error",{...}]）
func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	h.MessageSender.SendError(conn, "/", errors.New(errMsg))
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	for _, userID := range userIDs {
		if conn, ok := h.ConnectionManager.GetConnection(userID); ok {
			if err := h.MessageSender.SendEvent(conn, "/", "message", wsMsg); err != nil {
				return err
			}
		}
	}
	return nil
}

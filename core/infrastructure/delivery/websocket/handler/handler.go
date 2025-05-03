package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gorilla/websocket"
)

type Handler struct {
	ChatUseCase       *usecase.ChatUseCase
	ConnectionManager *connection.Manager
	connections       sync.Map // map[string]*websocket.Conn
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

func (h *Handler) HandleDisconnect(conn *websocket.Conn, reason string) {
	log.Println("disconnected:", conn.RemoteAddr(), reason)
	// Remove connection from map
	h.connections.Range(func(key, value interface{}) bool {
		if value.(*websocket.Conn) == conn {
			h.connections.Delete(key)
			return false
		}
		return true
	})
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
	if len(msg.Dst) > 5 && msg.Dst[:5] == "room:" {
		roomID := msg.Dst[5:]
		msg.Status = entity.StatusDelivered
		wsMsg := dto.FromEntity(msg).ToWSMessage()
		return h.ConnectionManager.BroadcastToRoom(wsMsg, roomID)
	}

	// Handle direct messages
	recipientID := msg.Dst
	if len(msg.Dst) > 2 && msg.Dst[1] == ':' {
		recipientID = msg.Dst[2:]
	}

	msg.Status = entity.StatusDelivered
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	data, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	if conn, ok := h.connections.Load(recipientID); ok {
		return conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, data)
	}

	// 接收方离线，更新为离线状态
	return h.ChatUseCase.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
}

// sendError 发送错误消息
func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	errResp := dto.WSMessage{
		Code: 400,
		Msg:  errMsg,
		Data: nil,
	}
	data, _ := json.Marshal(errResp)
	conn.WriteMessage(websocket.TextMessage, data)
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	data, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		if conn, ok := h.connections.Load(userID); ok {
			conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, data)
		}
	}
	return nil
}

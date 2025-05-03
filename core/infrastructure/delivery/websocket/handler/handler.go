package handler

import (
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/dto"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/infrastructure/delivery/websocket/connection"
	"cland.org/cland-chat-service/core/usecase"
	socketio "github.com/googollee/go-socket.io"
)

type Handler struct {
	Server            *socketio.Server
	ChatUseCase       *usecase.ChatUseCase
	ConnectionManager *connection.Manager
	connections       sync.Map // map[string]socketio.Conn
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CORS for preflight requests
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Let go-socket.io handle WebSocket and polling
	h.Server.ServeHTTP(w, r)
}

func (h *Handler) HandleMessage(conn socketio.Conn, data string) {
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

func (h *Handler) HandleError(conn socketio.Conn, err error) {
	log.Println("socket error:", err)
}

func (h *Handler) HandleDisconnect(conn socketio.Conn, reason string) {
	log.Println("disconnected:", conn.ID(), reason)
	// Remove connection from map
	h.connections.Range(func(key, value interface{}) bool {
		if value.(socketio.Conn) == conn {
			h.connections.Delete(key)
			return false
		}
		return true
	})
}

// processMessage 处理消息业务逻辑 (保持不变)
func (h *Handler) processMessage(conn socketio.Conn, msg entity.Message) error {
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

// pushMessage 推送消息给接收方 (修改为使用socket.io)
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

	if conn, ok := h.connections.Load(recipientID); ok {
		conn.(socketio.Conn).Emit("message", wsMsg)
	}

	// 接收方离线，更新为离线状态
	return h.ChatUseCase.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
}

// sendError 发送错误消息
func (h *Handler) sendError(conn socketio.Conn, errMsg string) {
	errResp := dto.WSMessage{
		Code: 400,
		Msg:  errMsg,
		Data: nil,
	}
	conn.Emit("error", errResp)
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	wsMsg := dto.FromEntity(msg).ToWSMessage()
	for _, userID := range userIDs {
		if conn, ok := h.connections.Load(userID); ok {
			conn.(socketio.Conn).Emit("message", wsMsg)
		}
	}
	return nil
}

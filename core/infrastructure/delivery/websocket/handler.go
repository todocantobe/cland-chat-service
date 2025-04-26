package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/usecase"
	socketio "github.com/googollee/go-socket.io"
)

type Handler struct {
	server      *socketio.Server
	chatUseCase *usecase.ChatUseCase
	connections sync.Map // map[string]socketio.Conn
}

func NewHandler(chatUseCase *usecase.ChatUseCase) *Handler {
	server := socketio.NewServer(nil)
	h := &Handler{
		server:      server,
		chatUseCase: chatUseCase,
	}

	// Setup socket.io event handlers
	server.OnConnect("/", func(conn socketio.Conn) error {
		log.Println("connected:", conn.ID())
		return nil
	})

	server.OnEvent("/", "auth", h.handleAuth)
	server.OnEvent("/", "message", h.handleMessage)
	server.OnError("/", h.handleError)
	server.OnDisconnect("/", h.handleDisconnect)

	return h
}

func (h *Handler) GetServer() *socketio.Server {
	return h.server
}

func (h *Handler) handleAuth(conn socketio.Conn, token string) {
	// TODO: Implement authentication
	userID := token // For now just use token as userID
	h.connections.Store(userID, conn)
	conn.Join(userID)
}

func (h *Handler) handleMessage(conn socketio.Conn, data string) {
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

func (h *Handler) handleError(conn socketio.Conn, err error) {
	log.Println("socket error:", err)
}

func (h *Handler) handleDisconnect(conn socketio.Conn, reason string) {
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
		if err := h.chatUseCase.SendMessage(ctx, &msg); err != nil {
			return err
		}
		return h.pushMessage(msg)
	case entity.MsgTypeAck:
		return h.chatUseCase.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusRead)
	default:
		return errors.New("unsupported message type")
	}
}

// pushMessage 推送消息给接收方 (修改为使用socket.io)
func (h *Handler) pushMessage(msg entity.Message) error {
	recipientID := msg.Dst
	if len(msg.Dst) > 2 && msg.Dst[1] == ':' {
		recipientID = msg.Dst[2:]
	}

	msg.Status = entity.StatusDelivered
	wsMsg := FromEntity(msg).ToWSMessage()

	if conn, ok := h.connections.Load(recipientID); ok {
		conn.(socketio.Conn).Emit("message", wsMsg)
	}

	// 接收方离线，更新为离线状态
	return h.chatUseCase.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
}

// sendError 发送错误消息
func (h *Handler) sendError(conn socketio.Conn, errMsg string) {
	errResp := WSMessage{
		Code: 400,
		Msg:  errMsg,
		Data: nil,
	}
	conn.Emit("error", errResp)
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	wsMsg := FromEntity(msg).ToWSMessage()
	for _, userID := range userIDs {
		if conn, ok := h.connections.Load(userID); ok {
			conn.(socketio.Conn).Emit("message", wsMsg)
		}
	}
	return nil
}

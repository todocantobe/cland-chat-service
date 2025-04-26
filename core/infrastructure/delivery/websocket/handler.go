package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/usecase"
	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader    websocket.Upgrader
	chatUseCase *usecase.ChatUseCase
	connections sync.Map // map[string]*websocket.Conn
}

func NewHandler(chatUseCase *usecase.ChatUseCase) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		chatUseCase: chatUseCase,
	}
}

// HandleConnection handles WebSocket connection with authentication
func (h *Handler) HandleConnection(conn *websocket.Conn, cid string) {

	// Store connection
	h.connections.Store(cid, conn)
	defer func() {
		h.connections.Delete(cid)
		conn.Close()
	}()

	for {
		// 读取消息
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}

		// 解析消息
		var msg entity.Message
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			h.sendError(conn, "invalid message format")
			continue
		}

		// 处理消息
		if err := h.processMessage(conn, msg); err != nil {
			h.sendError(conn, err.Error())
		}
	}
}

// processMessage 处理消息业务逻辑
func (h *Handler) processMessage(conn *websocket.Conn, msg entity.Message) error {
	ctx := context.Background()

	// 处理消息
	switch msg.MsgType {
	case entity.MsgTypeMessage, entity.MsgTypeNotification:
		if err := h.chatUseCase.SendMessage(ctx, &msg); err != nil {
			return err
		}
		// 推送消息给接收方
		return h.pushMessage(msg)
	case entity.MsgTypeAck:
		return h.chatUseCase.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusRead)
	default:
		return errors.New("unsupported message type")
	}
}

// pushMessage 推送消息给接收方
func (h *Handler) pushMessage(msg entity.Message) error {
	// 获取接收方连接
	conn, ok := h.connections.Load(msg.Dst)
	if !ok {
		// 接收方离线，更新为离线状态
		return h.chatUseCase.ProcessMessageStatus(context.Background(), msg.MsgID, entity.StatusOffline)
	}

	// 发送消息
	wsConn := conn.(*websocket.Conn)
	msg.Status = entity.StatusDelivered
	wsMsg := FromEntity(msg).ToWSMessage()

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	return wsConn.WriteMessage(websocket.TextMessage, msgBytes)
}

// sendError 发送错误消息
func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	errResp := WSMessage{
		Code: 400,
		Msg:  errMsg,
		Data: nil,
	}
	conn.WriteJSON(errResp)
}

// BroadcastMessage 广播消息给多个用户
func (h *Handler) BroadcastMessage(msg entity.Message, userIDs []string) error {
	for _, userID := range userIDs {
		if conn, ok := h.connections.Load(userID); ok {
			wsConn := conn.(*websocket.Conn)
			wsMsg := FromEntity(msg).ToWSMessage()
			if err := wsConn.WriteJSON(wsMsg); err != nil {
				return err
			}
		}
	}
	return nil
}

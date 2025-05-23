package usecase

import (
	"context"
	"errors"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/domain/repository"
)

// ChatUseCase 聊天用例
type ChatUseCase struct {
	messageRepo repository.MessageRepository
	SessionRepo repository.SessionRepository
	UserRepo    repository.UserRepository
}

// NewChatUseCase 创建聊天用例
func NewChatUseCase(
	messageRepo repository.MessageRepository,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) *ChatUseCase {
	return &ChatUseCase{
		messageRepo: messageRepo,
		SessionRepo: sessionRepo,
		UserRepo:    userRepo,
	}
}

// SendMessage 发送消息
func (uc *ChatUseCase) SendMessage(ctx context.Context, message *entity.Message) error {
	// 初始化消息时间戳
	if message.Ts == 0 {
		message.Ts = entity.StringTimestamp(time.Now().UnixNano() / int64(time.Millisecond))
	}

	// 处理不同类型的消息
	switch message.MsgType {
	case entity.MsgTypeMessage:
		return uc.handleChatMessage(ctx, message)
	case entity.MsgTypeNotification:
		return uc.handleNotification(ctx, message)
	case entity.MsgTypeAck:
		return uc.handleAck(ctx, message)
	default:
		return errors.New("invalid message type")
	}
}

// handleChatMessage 处理普通聊天消息
func (uc *ChatUseCase) handleChatMessage(ctx context.Context, message *entity.Message) error {
	// 设置初始状态
	message.Status = entity.StatusNew

	// 保存消息
	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return err
	}

	// 更新为已发送状态
	message.Status = entity.StatusSent
	return uc.messageRepo.UpdateStatus(ctx, message.MsgID, message.Status)
}

// handleNotification 处理通知消息
func (uc *ChatUseCase) handleNotification(ctx context.Context, message *entity.Message) error {
	// 初始化消息不需要会话检查
	if message.Content == "init" {
		message.Status = entity.StatusNew
		return uc.messageRepo.Create(ctx, message)
	}

	// 其他通知需要会话检查
	_, err := uc.SessionRepo.GetByID(ctx, message.SessionID)
	if err != nil {
		return err
	}

	message.Status = entity.StatusNew
	return uc.messageRepo.Create(ctx, message)
}

// handleAck 处理确认消息
func (uc *ChatUseCase) handleAck(ctx context.Context, message *entity.Message) error {
	// 获取原始消息
	original, err := uc.messageRepo.GetByID(ctx, message.MsgID)
	if err != nil {
		return err
	}

	// 验证状态转换是否有效
	switch original.Status {
	case entity.StatusDelivered:
		original.Status = entity.StatusRead
	case entity.StatusNew, entity.StatusSent:
		original.Status = entity.StatusDelivered
	default:
		return errors.New("invalid status transition")
	}

	// 更新消息状态
	return uc.messageRepo.UpdateStatus(ctx, original.MsgID, original.Status)
}

// GetSessionMessages 获取会话消息
func (uc *ChatUseCase) GetSessionMessages(ctx context.Context, sessionID string) ([]*entity.Message, error) {
	messages, err := uc.messageRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 将历史消息标记为已读状态
	for _, msg := range messages {
		if msg.Status == entity.StatusDelivered {
			msg.Status = entity.StatusRead
		} else if msg.Status == entity.StatusNew || msg.Status == entity.StatusSent {
			msg.Status = entity.StatusHistory
		}
	}

	return messages, nil
}

// CreateSession 创建会话
func (uc *ChatUseCase) CreateSession(ctx context.Context, userID string) (*entity.Session, error) {
	// 获取可用客服
	agents, err := uc.UserRepo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	// 分配客服
	agentID := ""
	if len(agents) > 0 {
		agentID = agents[0].ID
	}

	session := &entity.Session{
		CID:     userID,
		AgentId: agentID,
		Status:  "active",
	}

	if err := uc.SessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// CloseSession 关闭会话
func (uc *ChatUseCase) CloseSession(ctx context.Context, sessionID string) error {
	return uc.SessionRepo.UpdateStatus(ctx, sessionID, "closed")
}

// GetOfflineMessages 获取离线消息并更新状态
func (uc *ChatUseCase) GetOfflineMessages(ctx context.Context, userID string) ([]*entity.Message, error) {
	// Get all messages for user
	allMessages, err := uc.GetSessionMessages(ctx, userID)
	if err != nil {
		return nil, err
	}

	var messages []*entity.Message
	for _, msg := range allMessages {
		if msg.Status == entity.StatusOffline {
			// Update status to delivered
			if err := uc.ProcessMessageStatus(ctx, msg.MsgID, entity.StatusDelivered); err == nil {
				messages = append(messages, msg)
			}
		}
	}

	return messages, nil
}

// ProcessMessageStatus 处理消息状态更新
func (uc *ChatUseCase) ProcessMessageStatus(ctx context.Context, msgID string, newStatus uint8) error {
	// 获取消息
	message, err := uc.messageRepo.GetByID(ctx, msgID)
	if err != nil {
		return err
	}

	// 验证状态转换
	switch {
	case message.Status == entity.StatusNew && (newStatus == entity.StatusSent || newStatus == entity.StatusOffline):
	case message.Status == entity.StatusSent && (newStatus == entity.StatusDelivered || newStatus == entity.StatusOffline):
	case message.Status == entity.StatusDelivered && newStatus == entity.StatusRead:
	case message.Status == entity.StatusOffline && newStatus == entity.StatusDelivered:
	case newStatus == entity.StatusRecall: // 允许从多个状态撤回
	default:
		return errors.New("invalid status transition")
	}

	// 更新状态
	return uc.messageRepo.UpdateStatus(ctx, msgID, newStatus)
}

package usecase

import (
	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/domain/repository"
	"context"
)

// ChatUseCase 聊天用例
type ChatUseCase struct {
	messageRepo repository.MessageRepository
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
}

// NewChatUseCase 创建聊天用例
func NewChatUseCase(
	messageRepo repository.MessageRepository,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) *ChatUseCase {
	return &ChatUseCase{
		messageRepo: messageRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// SendMessage 发送消息
func (uc *ChatUseCase) SendMessage(ctx context.Context, message *entity.Message) error {
	// 检查会话是否存在
	_, err := uc.sessionRepo.GetByID(ctx, message.SessionID)
	if err != nil {
		return err
	}

	// 更新消息状态
	message.Status = "sent"

	// 保存消息
	return uc.messageRepo.Create(ctx, message)
}

// GetSessionMessages 获取会话消息
func (uc *ChatUseCase) GetSessionMessages(ctx context.Context, sessionID string) ([]*entity.Message, error) {
	return uc.messageRepo.GetBySessionID(ctx, sessionID)
}

// CreateSession 创建会话
func (uc *ChatUseCase) CreateSession(ctx context.Context, userID string) (*entity.Session, error) {
	// 获取可用客服
	agents, err := uc.userRepo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	// 分配客服
	agentID := ""
	if len(agents) > 0 {
		agentID = agents[0].ID
	}

	session := &entity.Session{
		UserID:  userID,
		AgentID: agentID,
		Status:  "active",
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// CloseSession 关闭会话
func (uc *ChatUseCase) CloseSession(ctx context.Context, sessionID string) error {
	return uc.sessionRepo.UpdateStatus(ctx, sessionID, "closed")
}

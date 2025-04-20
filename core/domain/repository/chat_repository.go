package repository

import (
	"cland.org/cland-chat-service/core/domain/entity"
	"context"
)

// MessageRepository 消息仓储接口
type MessageRepository interface {
	Create(ctx context.Context, message *entity.Message) error
	GetBySessionID(ctx context.Context, sessionID string) ([]*entity.Message, error)
	UpdateStatus(ctx context.Context, messageID uint, status string) error
}

// SessionRepository 会话仓储接口
type SessionRepository interface {
	Create(ctx context.Context, session *entity.Session) error
	GetByID(ctx context.Context, id string) (*entity.Session, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	ListActive(ctx context.Context) ([]*entity.Session, error)
}

// UserRepository 用户仓储接口
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*entity.User, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	ListAgents(ctx context.Context) ([]*entity.User, error)
}

package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
)

// MemoryMessageRepository 实现MessageRepository
type MemoryMessageRepository struct {
	store sync.Map
}

func NewMemoryMessageRepository() *MemoryMessageRepository {
	return &MemoryMessageRepository{}
}

// MemorySessionRepository 实现SessionRepository
type MemorySessionRepository struct {
	store sync.Map
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{}
}

// MemoryUserRepository 实现UserRepository
type MemoryUserRepository struct {
	store sync.Map
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{}
}

func (r *MemoryUserRepository) Create(ctx context.Context, user *entity.User) error {
	r.store.Store(user.ID, user)
	return nil
}

func (r *MemoryUserRepository) CreateOrUpdate(ctx context.Context, user *entity.User) error {
	r.store.Store(user.ID, user)
	return nil
}

// MessageRepository implementation
func (r *MemoryMessageRepository) Create(ctx context.Context, message *entity.Message) error {
	r.store.Store(message.MsgID, message)
	return nil
}

func (r *MemoryMessageRepository) GetByID(ctx context.Context, msgID string) (*entity.Message, error) {
	val, ok := r.store.Load(msgID)
	if !ok {
		return nil, ErrNotFound
	}
	return val.(*entity.Message), nil
}

func (r *MemoryMessageRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*entity.Message, error) {
	var messages []*entity.Message
	r.store.Range(func(_, value interface{}) bool {
		msg := value.(*entity.Message)
		if msg.SessionID == sessionID {
			messages = append(messages, msg)
		}
		return true
	})
	return messages, nil
}

func (r *MemoryMessageRepository) UpdateStatus(ctx context.Context, msgID string, status uint8) error {
	val, ok := r.store.Load(msgID)
	if !ok {
		return ErrNotFound
	}
	msg := val.(*entity.Message)
	msg.Status = status
	msg.Ts = entity.StringTimestamp(time.Now().UnixNano() / int64(time.Millisecond))
	r.store.Store(msgID, msg)
	return nil
}

// SessionRepository implementation
func (r *MemorySessionRepository) Create(ctx context.Context, session *entity.Session) error {
	r.store.Store(session.ID, session)
	return nil
}

func (r *MemorySessionRepository) GetByID(ctx context.Context, id string) (*entity.Session, error) {
	val, ok := r.store.Load(id)
	if !ok {
		return nil, ErrNotFound
	}
	return val.(*entity.Session), nil
}

func (r *MemorySessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	val, ok := r.store.Load(id)
	if !ok {
		return ErrNotFound
	}
	session := val.(*entity.Session)
	session.Status = status
	r.store.Store(id, session)
	return nil
}

func (r *MemorySessionRepository) ListActive(ctx context.Context) ([]*entity.Session, error) {
	var sessions []*entity.Session
	r.store.Range(func(_, value interface{}) bool {
		session := value.(*entity.Session)
		if session.Status == "active" {
			sessions = append(sessions, session)
		}
		return true
	})
	return sessions, nil
}

// UserRepository implementation
func (r *MemoryUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	val, ok := r.store.Load(id)
	if !ok {
		return nil, ErrNotFound
	}
	return val.(*entity.User), nil
}

func (r *MemoryUserRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	val, ok := r.store.Load(id)
	if !ok {
		return ErrNotFound
	}
	user := val.(*entity.User)
	user.Status = status
	r.store.Store(id, user)
	return nil
}

func (r *MemoryUserRepository) ListAgents(ctx context.Context) ([]*entity.User, error) {
	var agents []*entity.User
	r.store.Range(func(_, value interface{}) bool {
		user := value.(*entity.User)
		if user.Role == "agent" {
			agents = append(agents, user)
		}
		return true
	})
	return agents, nil
}

var ErrNotFound = errors.New("not found")

package output

import (
	domain_repository "cland.org/cland-chat-service/core/domain/repository"
	infra_repository "cland.org/cland-chat-service/core/infrastructure/repository"
)

// RepositoryAdapter 仓储适配器
type RepositoryAdapter struct {
	messageRepo domain_repository.MessageRepository
	sessionRepo domain_repository.SessionRepository
	userRepo    domain_repository.UserRepository
}

// NewRepositoryAdapter 创建仓储适配器
func NewRepositoryAdapter(
	messageRepo domain_repository.MessageRepository,
	sessionRepo domain_repository.SessionRepository,
	userRepo domain_repository.UserRepository,
) *RepositoryAdapter {
	return &RepositoryAdapter{
		messageRepo: messageRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// GetMessageRepository 获取消息仓储
func (a *RepositoryAdapter) GetMessageRepository() domain_repository.MessageRepository {
	return a.messageRepo
}

// GetSessionRepository 获取会话仓储
func (a *RepositoryAdapter) GetSessionRepository() domain_repository.SessionRepository {
	return a.sessionRepo
}

// GetUserRepository 获取用户仓储
func (a *RepositoryAdapter) GetUserRepository() domain_repository.UserRepository {
	return a.userRepo
}

// RepositoryFactory 仓储工厂
type RepositoryFactory struct{}

// NewRepositoryFactory 创建仓储工厂
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// CreateSQLiteRepositories 创建SQLite仓储实例
func (f *RepositoryFactory) CreateSQLiteRepositories(dbPath string) (
	domain_repository.MessageRepository,
	domain_repository.SessionRepository,
	domain_repository.UserRepository,
	error,
) {
	_, messageRepo, sessionRepo, userRepo, err := infra_repository.NewSQLiteRepository(dbPath)
	if err != nil {
		return nil, nil, nil, err
	}

	return messageRepo, sessionRepo, userRepo, nil
}

// CreateMemoryRepositories 创建内存仓储实例
func (f *RepositoryFactory) CreateMemoryRepositories() (
	domain_repository.MessageRepository,
	domain_repository.SessionRepository,
	domain_repository.UserRepository,
) {
	messageRepo := infra_repository.NewMemoryMessageRepository()
	sessionRepo := infra_repository.NewMemorySessionRepository()
	userRepo := infra_repository.NewMemoryUserRepository()

	return messageRepo, sessionRepo, userRepo
}

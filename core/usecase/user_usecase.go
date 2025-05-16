package usecase

import (
	"context"
	"time"

	"cland.org/cland-chat-service/common/utils"
	"cland.org/cland-chat-service/core/domain/entity"
	"cland.org/cland-chat-service/core/domain/repository"
)

type UserUseCase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewUserUseCase(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) *UserUseCase {
	return &UserUseCase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

type InitUserResponse struct {
	SessionID    string
	SubSessionID string
	Token        string
	ClandCID     string
}

func (uc *UserUseCase) InitUser(ctx context.Context, existingCID string) (*InitUserResponse, error) {
	// Generate or validate CID
	var clandCID string
	if utils.IsValidClandCID(existingCID) {
		clandCID = existingCID
	} else {
		clandCID = utils.GenerateClandCID()
	}

	// Create or update user
	user := &entity.User{
		ID:         clandCID,
		Username:   "guest_" + clandCID[1:7],
		Role:       "customer",
		Status:     "online",
		LastActive: time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Create new session
	sessionID := utils.GenerateSessionID()
	subSessionID := utils.GenerateSubSessionID()

	session := &entity.Session{
		ID:           sessionID,
		UserID:       clandCID,
		SubSessionID: subSessionID,
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(clandCID)
	if err != nil {
		return nil, err
	}

	return &InitUserResponse{
		SessionID:    sessionID,
		SubSessionID: subSessionID,
		Token:        token,
		ClandCID:     clandCID,
	}, nil
}

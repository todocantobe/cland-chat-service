package input

import (
	"context"

	"cland.org/cland-chat-service/core/application"
)

// UserController 用户控制器接口
type UserController interface {
	// InitUser 初始化用户
	InitUser(ctx context.Context, existingCID string) (*application.InitUserResponse, error)
}

// HTTPUserController HTTP用户控制器实现
type HTTPUserController struct {
	userService *application.UserService
}

// NewHTTPUserController 创建HTTP用户控制器
func NewHTTPUserController(userService *application.UserService) *HTTPUserController {
	return &HTTPUserController{
		userService: userService,
	}
}

// InitUser 初始化用户
func (c *HTTPUserController) InitUser(ctx context.Context, existingCID string) (*application.InitUserResponse, error) {
	return c.userService.InitUser(ctx, existingCID)
}

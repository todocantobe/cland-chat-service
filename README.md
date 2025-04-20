# Cland Chat Service

一个基于Go语言的客服聊天系统，采用简洁架构(Clean Architecture)设计。

## 项目结构

```
cland-chat-service/
├── internal/              # 内部包
│   ├── domain/           # 领域层
│   │   ├── entity/       # 领域实体
│   │   │   └── chat.go   # 聊天相关实体
│   │   └── repository/   # 仓储接口
│   │       └── chat_repository.go  # 聊天仓储接口
│   ├── usecase/          # 用例层
│   │   └── chat_usecase.go  # 聊天用例实现
│   └── delivery/         # 交付层
│       ├── http/         # HTTP处理
│       │   └── chat_handler.go  # HTTP处理器
│       └── websocket/    # WebSocket处理
│           └── chat_handler.go  # WebSocket处理器
├── pkg/                   # 公共包
│   ├── config/           # 配置
│   ├── logger/           # 日志
│   └── utils/            # 工具函数
├── api/                   # API文档
├── scripts/              # 脚本文件
├── main.go               # 程序入口
└── go.mod                # Go模块文件
```

## 核心功能

- 实时聊天
- 会话管理
- 消息存储
- 客服分配
- WebSocket通信

## API接口

### HTTP API

- `POST /api/chat/sessions` - 创建会话
- `POST /api/chat/sessions/:id/messages` - 发送消息
- `GET /api/chat/sessions/:id/messages` - 获取消息
- `POST /api/chat/sessions/:id/close` - 关闭会话

### WebSocket API

- `GET /ws` - WebSocket连接
  - 参数:
    - `user_id` - 用户ID
    - `session_id` - 会话ID

## 技术栈

### 后端技术

- **Go 1.21**
  - 高性能、并发支持
  - 标准库丰富
  - 跨平台编译

- **Gin Web框架**
  - 高性能HTTP框架
  - 中间件支持
  - 路由分组
  - 参数绑定和验证

- **GORM ORM框架**
  - 支持多种数据库
  - 链式操作
  - 事务支持
  - 关联查询

- **Redis**
  - 消息缓存
  - 会话状态管理
  - 分布式锁
  - 发布订阅

- **WebSocket**
  - 实时双向通信
  - 心跳检测
  - 连接管理
  - 消息广播

### 开发工具

- **Git** - 版本控制
- **Docker** - 容器化部署
- **Make** - 构建工具
- **Swagger** - API文档
- **GoLand/VSCode** - IDE支持

### 测试工具

- **Go Test** - 单元测试
- **GoMock** - 接口模拟
- **Testify** - 断言库
- **Ginkgo** - BDD测试框架

### 监控和日志

- **Prometheus** - 指标监控
- **Grafana** - 可视化面板
- **ELK Stack** - 日志收集
- **Zap** - 高性能日志库

### 部署相关

- **Docker Compose** - 容器编排
- **Kubernetes** - 容器编排
- **Nginx** - 反向代理
- **Let's Encrypt** - SSL证书

## 架构设计

项目采用简洁架构设计，分为以下层次：

1. 领域层(Domain)
   - 包含核心业务实体和规则
   - 定义仓储接口

2. 用例层(UseCase)
   - 实现具体业务逻辑
   - 协调领域对象

3. 交付层(Delivery)
   - 处理HTTP和WebSocket请求
   - 实现API接口

4. 基础设施层(Infrastructure)
   - 实现仓储接口
   - 处理数据库操作

## 开发环境

1. 安装依赖
```bash
go mod download
```

2. 运行服务
```bash
go run main.go
```

## 配置说明

服务默认运行在 `:8080` 端口，可通过环境变量修改：

- `PORT` - 服务端口
- `REDIS_ADDR` - Redis地址
- `DB_DSN` - 数据库连接字符串

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request
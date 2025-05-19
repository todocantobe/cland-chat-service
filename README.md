# Cland Chat Service

基于Go语言的客服聊天系统，采用简洁架构(Clean Architecture)设计。

## 项目结构 (更新)

```
cland-chat-service/
├── core/                 # 核心业务代码
│   ├── domain/           # 领域层
│   │   ├── entity/       # 领域实体
│   │   └── repository/   # 仓储接口
│   ├── usecase/          # 用例层(业务逻辑)
│   └── infrastructure/   # 基础设施层
│       ├── delivery/     # 交付层(HTTP/WebSocket)
│       └── repository/   # 仓储实现(Memory/SQLite)
├── common/               # 公共组件
│   ├── constants/        # 常量定义
│   ├── errors/           # 错误定义
│   └── utils/            # 工具函数
├── conf/                 # 配置文件
├── docs/                 # 文档
├── http/                 # HTTP测试脚本
├── sql/                  # SQL初始化脚本
├── main.go               # 程序入口
└── go.mod                # Go模块文件
```

## 架构设计

### 简洁架构分层

1. **领域层(Domain)**
   - 定义核心业务实体(Entity)和仓储接口(Repository)
   - 包含业务规则和领域逻辑
   - 示例: `core/domain/entity/chat.go`, `core/domain/repository/chat_repository.go`

2. **用例层(UseCase)**
   - 实现具体业务逻辑
   - 协调领域对象和仓储
   - 示例: `core/usecase/chat_usecase.go`

3. **基础设施层(Infrastructure)**
   - 实现仓储接口(SQLite/Memory)
   - 处理外部交互(HTTP/WebSocket)
   - 示例: `core/infrastructure/repository/sqlite_repository.go`

4. **交付层(Delivery)**
   - 处理外部请求和响应
   - 转换DTO和领域对象
   - 示例: `core/infrastructure/delivery/http/handler/chat_handler.go`

### 依赖关系

```
Delivery → UseCase → Domain
            ↑
Infrastructure → Domain
```

## 核心功能

- 实时聊天(WebSocket)
- 会话管理
- 消息存储(SQLite/Memory)
- 客服分配
- REST API接口

## 技术栈

### 后端技术

- **Go 1.24**
  - 高性能、并发支持
  - 标准库丰富

- **Gin Web框架**
  - 高性能HTTP框架
  - 中间件支持

- **SQLite**
  - 轻量级嵌入式数据库
  - 支持事务
  - 本地存储方案

- **WebSocket**
  - 实时双向通信
  - 心跳检测
  - 连接管理

### 开发工具

- **Git** - 版本控制
- **Docker** - 容器化部署
- **Make** - 构建工具
- **Swagger** - API文档

## 如何使用

1. 安装依赖
```bash
go mod download
```

2. 初始化数据库
```bash
sqlite3 chat.db < sql/init.sql
```

3. 运行服务
```bash
go run main.go
```

## 配置说明

配置文件位于 `conf/` 目录:

- `app.ini` - 应用配置
- `config.yaml` - 详细配置

环境变量覆盖:
- `PORT` - 服务端口(默认8080)
- `DB_PATH` - SQLite数据库路径(默认chat.db)

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

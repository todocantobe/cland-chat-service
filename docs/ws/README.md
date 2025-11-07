# WebSocket 测试客户端

## 安装依赖
```bash
npm install
```

## 测试用例

### 基础连接测试
```bash
node test-ws-client.js --serverUrl=http://localhost:8081 --namespace=/ --eventName=message --eventData='{"msgType":1,"sessionId":"test-session","msgId":"msg-001","src":"U:user_001","dst":"room:test-room","content":"Hello World","contentType":1,"ts":"1745690716604","status":1}' --cland-cid=1
```

### 简化连接测试
```bash
node test-ws-client.js --cland-cid=1
```

## 客户端命令说明

连接成功后，支持以下交互命令：

- `send [事件名] [数据]` - 发送事件（自动附加 cland-cid）
- `join [房间名]` - 加入房间
- `leave [房间名]` - 离开房间
- `broadcast [房间名] [事件名] [数据]` - 广播事件（自动附加 cland-cid）
- `set cland-cid [值]` - 动态修改 cland-cid（如：set cland-cid=2）
- `help` - 显示消息格式帮助
- `exit` - 退出

## 消息格式说明

消息需要符合以下 JSON 格式：
```json
{
  "msgType": 1,
  "sessionId": "test-session",
  "msgId": "msg-001",
  "src": "U:user_001",
  "dst": "room:test-room",
  "content": "Hello World",
  "contentType": 1,
  "ts": "1745690716604",
  "status": 1
}
```

### 字段说明
- `msgType`: 1=普通消息, 2=通知, 3=确认
- `sessionId`: 会话ID
- `msgId`: 消息ID
- `src`: 发送者 (U:用户, A:客服, S:系统)
- `dst`: 接收者 (用户ID 或 room:房间名)
- `content`: 消息内容
- `contentType`: 1=文本, 2=图片, 3=文件, 520=转人工
- `ts`: 时间戳 (Unix毫秒)
- `status`: 1=新建, 2=历史, 3=离线, 4=撤回, 5=已发送, 6=已送达, 7=已读

## 注意事项
- 连接时必须提供 `cland-cid` 参数，否则服务器会拒绝连接
- 服务器默认运行在 `localhost:8081`
- 支持动态修改 cland-cid，无需重新连接

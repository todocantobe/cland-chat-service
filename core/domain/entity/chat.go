package entity

import "time"

// 消息类型枚举
const (
	MsgTypeMessage = 1 // 普通消息
	MsgTypeNotification = 2 // 通知
	MsgTypeAck = 3 // 确认
)

// 内容类型枚举
const (
	ContentTypeText = 1 // 文本
	ContentTypeImage = 2 // 图片
	ContentTypeFile = 3 // 文件
)

// 消息状态枚举
const (
	StatusNew = 1 // 新建
	StatusHistory = 2 // 历史消息
	StatusOffline = 3 // 离线
	StatusRecall = 4 // 撤回
	StatusSent = 5 // 已发送
	StatusDelivered = 6 // 已送达
	StatusRead = 7 // 已读
)

// Message 消息实体
type Message struct {
	MsgType     uint8                  `json:"msgType"` // 1=MSG, 2=NTF, 3=ACK
	SessionID   string                 `json:"sessionId"`
	MsgID       string                 `json:"msgId"`
	Src         string                 `json:"src"` // U:user_xxx, A:agent_xxx, S:system, UA:admin_xxx
	Dst         string                 `json:"dst"`
	Content     string                 `json:"content"`
	ContentType uint8                  `json:"contentType"` // 1=TEXT, 2=IMAGE, 3=FILE
	Ts          string                 `json:"ts"` // ISO8601格式时间戳
	Status      uint8                  `json:"status"` // 1=NEW, ..., 7=READ
	Ext         map[string]interface{} `json:"ext"` // 扩展字段
}

// Session 会话实体
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AgentID   string    `json:"agentId"`
	Status    string    `json:"status"` // active, closed
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// User 用户实体
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"` // customer, agent, admin
	Status   string `json:"status"` // online, offline, busy
}

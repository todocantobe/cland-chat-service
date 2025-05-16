package entity

import (
	"encoding/json"
	"strconv"
	"time"
)

// Message type enum values and type
const (
	MsgTypeMessage      = 1 // 普通消息
	MsgTypeNotification = 2 // 通知
	MsgTypeAck          = 3 // 确认
)

type MsgType uint8

// Content type enum values and type
const (
	ContentTypeText  = 1 // 文本
	ContentTypeImage = 2 // 图片
	ContentTypeFile  = 3 // 文件
)

type ContentType uint8

// Message status enum values and type
const (
	StatusNew       = 1 // 新建
	StatusHistory   = 2 // 历史消息
	StatusOffline   = 3 // 离线
	StatusRecall    = 4 // 撤回
	StatusSent      = 5 // 已发送
	StatusDelivered = 6 // 已送达
	StatusRead      = 7 // 已读
)

type Status uint8

// StringTimestamp is a custom type for parsing string timestamps into int64
type StringTimestamp int64

// UnmarshalJSON implements custom JSON unmarshaling for StringTimestamp
func (st *StringTimestamp) UnmarshalJSON(data []byte) error {
	// Handle null or empty values
	if string(data) == "null" || string(data) == `""` {
		*st = 0
		return nil
	}

	// Try to parse as a string first (e.g., "1745690716604")
	var str string
	if err := json.Unmarshal(data, &str); err == nil && str != "" {
		ts, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		*st = StringTimestamp(ts)
		return nil
	}

	// Try to parse as a number (e.g., 1745690716604)
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*st = StringTimestamp(num)
	return nil
}

// MarshalJSON implements custom JSON marshaling for StringTimestamp
func (st StringTimestamp) MarshalJSON() ([]byte, error) {
	// Convert int64 to string (e.g., 1745690716604 -> "1745690716604")
	str := strconv.FormatInt(int64(st), 10)
	return json.Marshal(str)
}

// Message 消息实体
type Message struct {
	MsgType      uint8                  `json:"msgType"` // 1=MSG, 2=NTF, 3=ACK
	SessionID    string                 `json:"sessionId"`
	SubSessionID string                 `json:"subSessionId"`
	MsgID        string                 `json:"msgId"`
	Src          string                 `json:"src"` // U:user_xxx, A:agent_xxx, S:system, UA:admin_xxx
	Dst          string                 `json:"dst"`
	Content      string                 `json:"content"`
	ContentType  uint8                  `json:"contentType"` // 1=TEXT, 2=IMAGE, 3=FILE
	Ts           StringTimestamp        `json:"ts"`          // Unix毫秒时间戳
	Status       uint8                  `json:"status"`      // 1=NEW, ..., 7=READ
	Ext          map[string]interface{} `json:"ext"`         // 扩展字段
}

// Session 会话实体
type Session struct {
	ID           string    `json:"id"`
	SubSessionID string    `json:"subSessionId"`
	UserID       string    `json:"userId"`
	AgentID      string    `json:"agentId"`
	Status       string    `json:"status"` // active, closed
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// User 用户实体
type User struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`   // customer, agent, admin
	Status     string `json:"status"` // online, offline, busy
	LastActive time.Time
}

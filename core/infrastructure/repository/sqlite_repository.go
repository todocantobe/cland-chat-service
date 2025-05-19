package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"cland.org/cland-chat-service/core/domain/entity"
	_ "github.com/mattn/go-sqlite3"
)

// Repository interfaces
type MessageRepository interface {
	CreateMessage(ctx context.Context, message *entity.Message) error
	GetMessageByID(ctx context.Context, msgID string) (*entity.Message, error)
	GetMessagesBySessionID(ctx context.Context, sessionID string) ([]*entity.Message, error)
	UpdateMessageStatus(ctx context.Context, msgID string, status uint8) error
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *entity.Session) error
	GetSessionByID(ctx context.Context, id string) (*entity.Session, error)
	UpdateSessionStatus(ctx context.Context, id string, status string) error
	ListActiveSessions(ctx context.Context) ([]*entity.Session, error)
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
	UpdateUserStatus(ctx context.Context, id string, status string) error
	ListAgentUsers(ctx context.Context) ([]*entity.User, error)
}

// Repository DTOs
type MessageDTO struct {
	MsgID       string
	SessionID   string
	MsgType     uint8
	Src         string
	Dst         string
	Content     string
	ContentType uint8
	Ts          int64
	Status      uint8
	Ext         []byte
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SessionDTO struct {
	ID        string
	CID       string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	CreatedBy string
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserDTO struct {
	ID        string
	UID       string
	Username  string
	Query     string
	Role      string
	Status    string
	CreatedBy string
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SQLiteRepository struct {
	db *sql.DB
}

func toMessageDTO(msg *entity.Message) MessageDTO {
	ext, _ := json.Marshal(msg.Ext)
	return MessageDTO{
		MsgID:       msg.MsgID,
		SessionID:   msg.SessionID,
		MsgType:     msg.MsgType,
		Src:         msg.Src,
		Dst:         msg.Dst,
		Content:     msg.Content,
		ContentType: msg.ContentType,
		Ts:          int64(msg.Ts),
		Status:      msg.Status,
		Ext:         ext,
		CreatedBy:   msg.CreatedBy,
		UpdatedBy:   msg.UpdatedBy,
		CreatedAt:   msg.CreatedAt,
		UpdatedAt:   msg.UpdatedAt,
	}
}

func toMessageEntity(dto MessageDTO) *entity.Message {
	var ext map[string]interface{}
	if len(dto.Ext) > 0 {
		json.Unmarshal(dto.Ext, &ext)
	}
	return &entity.Message{
		MsgID:       dto.MsgID,
		SessionID:   dto.SessionID,
		MsgType:     dto.MsgType,
		Src:         dto.Src,
		Dst:         dto.Dst,
		Content:     dto.Content,
		ContentType: dto.ContentType,
		Ts:          entity.StringTimestamp(dto.Ts),
		Status:      dto.Status,
		Ext:         ext,
		CreatedBy:   dto.CreatedBy,
		UpdatedBy:   dto.UpdatedBy,
		CreatedAt:   dto.CreatedAt,
		UpdatedAt:   dto.UpdatedAt,
	}
}

func toSessionDTO(session *entity.Session) SessionDTO {
	return SessionDTO{
		ID:        session.ID,
		CID:       session.CID,
		StartTime: session.StartTime,
		EndTime:   session.EndTime,
		Status:    session.Status,
		CreatedBy: session.CreatedBy,
		UpdatedBy: session.UpdatedBy,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

func toSessionEntity(dto SessionDTO) *entity.Session {
	return &entity.Session{
		ID:        dto.ID,
		CID:       dto.CID,
		StartTime: dto.StartTime,
		EndTime:   dto.EndTime,
		Status:    dto.Status,
		CreatedBy: dto.CreatedBy,
		UpdatedBy: dto.UpdatedBy,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}
}

func toUserDTO(user *entity.User) UserDTO {
	return UserDTO{
		ID:        user.ID,
		UID:       user.UID,
		Username:  user.Username,
		Query:     user.Query,
		Role:      user.Role,
		Status:    user.Status,
		CreatedBy: user.CreatedBy,
		UpdatedBy: user.UpdatedBy,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func toUserEntity(dto UserDTO) *entity.User {
	return &entity.User{
		ID:        dto.ID,
		UID:       dto.UID,
		Username:  dto.Username,
		Query:     dto.Query,
		Role:      dto.Role,
		Status:    dto.Status,
		CreatedBy: dto.CreatedBy,
		UpdatedBy: dto.UpdatedBy,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SQLiteRepository{db: db}, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// Interface implementation checks
var (
	_ MessageRepository = (*SQLiteRepository)(nil)
	_ SessionRepository = (*SQLiteRepository)(nil)
	_ UserRepository    = (*SQLiteRepository)(nil)
)

// MessageRepository implementation
func (r *SQLiteRepository) CreateMessage(ctx context.Context, message *entity.Message) error {
	query := `INSERT INTO t_chat_message 
		(msg_id, session_id, msg_type, src, dst, content, content_type, ts, status, ext, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	dto := toMessageDTO(message)
	_, err := r.db.ExecContext(ctx, query,
		dto.MsgID,
		dto.SessionID,
		dto.MsgType,
		dto.Src,
		dto.Dst,
		dto.Content,
		dto.ContentType,
		dto.Ts,
		dto.Status,
		dto.Ext,
		dto.CreatedBy,
		dto.UpdatedBy,
	)
	return err
}

func (r *SQLiteRepository) GetMessageByID(ctx context.Context, msgID string) (*entity.Message, error) {
	query := `SELECT 
		msg_id, session_id, msg_type, src, dst, content, content_type, ts, status, ext, 
		created_by, updated_by, created_at, updated_at
		FROM t_chat_message WHERE msg_id = ? AND is_deleted = 0`

	row := r.db.QueryRowContext(ctx, query, msgID)
	var dto MessageDTO
	err := row.Scan(
		&dto.MsgID,
		&dto.SessionID,
		&dto.MsgType,
		&dto.Src,
		&dto.Dst,
		&dto.Content,
		&dto.ContentType,
		&dto.Ts,
		&dto.Status,
		&dto.Ext,
		&dto.CreatedBy,
		&dto.UpdatedBy,
		&dto.CreatedAt,
		&dto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toMessageEntity(dto), nil
}

func (r *SQLiteRepository) GetMessagesBySessionID(ctx context.Context, sessionID string) ([]*entity.Message, error) {
	query := `SELECT 
		msg_id, session_id, msg_type, src, dst, content, content_type, ts, status, ext, 
		created_by, updated_by, created_at, updated_at
		FROM t_chat_message WHERE session_id = ? AND is_deleted = 0
		ORDER BY ts ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*entity.Message
	for rows.Next() {
		var dto MessageDTO
		err := rows.Scan(
			&dto.MsgID,
			&dto.SessionID,
			&dto.MsgType,
			&dto.Src,
			&dto.Dst,
			&dto.Content,
			&dto.ContentType,
			&dto.Ts,
			&dto.Status,
			&dto.Ext,
			&dto.CreatedBy,
			&dto.UpdatedBy,
			&dto.CreatedAt,
			&dto.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, toMessageEntity(dto))
	}
	return messages, nil
}

func (r *SQLiteRepository) UpdateMessageStatus(ctx context.Context, msgID string, status uint8) error {
	query := `UPDATE t_chat_message 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE msg_id = ?`

	_, err := r.db.ExecContext(ctx, query, status, msgID)
	return err
}

func (r *SQLiteRepository) CreateSession(ctx context.Context, session *entity.Session) error {
	query := `INSERT INTO t_session 
		(session_id, cid, start_time, end_time, status, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	dto := toSessionDTO(session)
	_, err := r.db.ExecContext(ctx, query,
		dto.ID,
		dto.CID,
		dto.StartTime,
		dto.EndTime,
		dto.Status,
		dto.CreatedBy,
		dto.UpdatedBy,
	)
	return err
}

func (r *SQLiteRepository) GetSessionByID(ctx context.Context, id string) (*entity.Session, error) {
	query := `SELECT 
		session_id, cid, start_time, end_time, status, 
		created_by, updated_by, created_at, updated_at
		FROM t_session WHERE session_id = ? AND is_deleted = 0`

	row := r.db.QueryRowContext(ctx, query, id)
	var dto SessionDTO
	err := row.Scan(
		&dto.ID,
		&dto.CID,
		&dto.StartTime,
		&dto.EndTime,
		&dto.Status,
		&dto.CreatedBy,
		&dto.UpdatedBy,
		&dto.CreatedAt,
		&dto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toSessionEntity(dto), nil
}

func (r *SQLiteRepository) UpdateSessionStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE t_session 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE session_id = ?`

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *SQLiteRepository) ListActiveSessions(ctx context.Context) ([]*entity.Session, error) {
	query := `SELECT 
		session_id, cid, start_time, end_time, status, 
		created_by, updated_by, created_at, updated_at
		FROM t_session WHERE status = 'active' AND is_deleted = 0`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*entity.Session
	for rows.Next() {
		var dto SessionDTO
		err := rows.Scan(
			&dto.ID,
			&dto.CID,
			&dto.StartTime,
			&dto.EndTime,
			&dto.Status,
			&dto.CreatedBy,
			&dto.UpdatedBy,
			&dto.CreatedAt,
			&dto.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, toSessionEntity(dto))
	}
	return sessions, nil
}

func (r *SQLiteRepository) CreateUser(ctx context.Context, user *entity.User) error {
	query := `INSERT INTO t_user 
		(cid, uid, query, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?)`

	dto := toUserDTO(user)
	_, err := r.db.ExecContext(ctx, query,
		dto.ID,
		dto.UID,
		dto.Query,
		dto.CreatedBy,
		dto.UpdatedBy,
	)
	return err
}

func (r *SQLiteRepository) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	query := `SELECT 
		cid, uid, query, 
		created_by, updated_by, created_at, updated_at
		FROM t_user WHERE cid = ? AND is_deleted = 0`

	row := r.db.QueryRowContext(ctx, query, id)
	var dto UserDTO
	err := row.Scan(
		&dto.ID,
		&dto.UID,
		&dto.Query,
		&dto.CreatedBy,
		&dto.UpdatedBy,
		&dto.CreatedAt,
		&dto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(dto), nil
}

func (r *SQLiteRepository) UpdateUserStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE t_user 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE cid = ?`

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *SQLiteRepository) ListAgentUsers(ctx context.Context) ([]*entity.User, error) {
	// Note: This implementation assumes agents are identified by a role field
	// which isn't in the current schema. Would need schema modification.
	return nil, errors.New("not implemented - requires schema changes")
}

// Remove duplicate ErrNotFound since it's already defined in memory_repository.go

-- Table: t_user
CREATE TABLE t_user (
    cid VARCHAR(50) NOT NULL,
    uid VARCHAR(50) NOT NULL,
    query TEXT,
    is_deleted INTEGER DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    updated_by VARCHAR(50) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (cid),
    UNIQUE (uid)
);
CREATE INDEX idx_t_user_created_at ON t_user(created_at);

-- Table: t_session
CREATE TABLE t_session (
    session_id VARCHAR(50) NOT NULL,
    cid VARCHAR(50) NOT NULL,
    start_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME,
    status INTEGER NOT NULL DEFAULT 1,
    is_deleted INTEGER DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    updated_by VARCHAR(50) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (session_id),
    FOREIGN KEY (cid) REFERENCES t_user(cid) ON DELETE CASCADE
);
CREATE INDEX idx_t_session_cid ON t_session(cid);
CREATE INDEX idx_t_session_start_time ON t_session(start_time);

-- Table: t_chat_message
CREATE TABLE t_chat_message (
    msg_id VARCHAR(50) NOT NULL,
    session_id VARCHAR(50) NOT NULL,
    msg_type INTEGER NOT NULL,
    src VARCHAR(50) NOT NULL,
    dst VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    content_type INTEGER NOT NULL,
    ts DATETIME NOT NULL,
    status INTEGER NOT NULL DEFAULT 1,
    ext TEXT,
    is_deleted INTEGER DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    updated_by VARCHAR(50) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (msg_id),
    FOREIGN KEY (session_id) REFERENCES t_session(session_id) ON DELETE CASCADE
);
CREATE INDEX idx_t_chat_message_session_id ON t_chat_message(session_id);
CREATE INDEX idx_t_chat_message_ts ON t_chat_message(ts);
CREATE INDEX idx_t_chat_message_status ON t_chat_message(status);
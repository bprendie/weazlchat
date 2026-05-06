package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
)

type Store struct {
	db       *sql.DB
	key      []byte
	unlocked bool
}

type Session struct {
	ID           string
	Title        string
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Message struct {
	ID         int64
	SessionID  string
	Role       string
	Content    string
	ToolCalls  string // JSON-encoded tool calls (for assistant messages)
	ToolCallID string // Tool call ID (for tool result messages)
	CreatedAt  time.Time
}

type WorkspaceSave struct {
	ID        int64
	Name      string
	SessionID string
	CreatedAt time.Time
}

type Memory struct {
	Key       string
	Value     string
	Tags      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func Open(path string) (*Store, error) {
	if err := mkdirFor(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate() error {
	stmts := []string{
		`create table if not exists vault (
			id integer primary key check (id = 1),
			password_hash text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists sessions (
			id text primary key,
			title text not null,
			provider text not null,
			model text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists messages (
			id integer primary key autoincrement,
			session_id text not null references sessions(id) on delete cascade,
			role text not null,
			content text not null,
			tool_calls text,
			tool_call_id text,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists workspace_saves (
			id integer primary key autoincrement,
			name text not null,
			session_id text not null references sessions(id) on delete cascade,
			snapshot text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists memories (
			key text primary key,
			value text not null,
			tags text,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create index if not exists idx_messages_session on messages(session_id, id)`,
		`create index if not exists idx_sessions_updated on sessions(updated_at desc)`,
		`create index if not exists idx_memories_updated on memories(updated_at desc)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := s.ensureColumn("sessions", "input_tokens", "integer not null default 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("sessions", "output_tokens", "integer not null default 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("messages", "tool_calls", "text"); err != nil {
		return err
	}
	if err := s.ensureColumn("messages", "tool_call_id", "text"); err != nil {
		return err
	}
	return nil
}

func (s *Store) HasVault() (bool, error) {
	var count int
	if err := s.db.QueryRow(`select count(*) from vault where id = 1`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateVault(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`insert into vault (id, password_hash) values (1, ?)`, string(hash))
	if err != nil {
		return err
	}
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlock(password string) error {
	var hash string
	if err := s.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errors.New("bad database password")
	}
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlocked() bool {
	return s.unlocked
}

func (s *Store) CreateSession(id, title, provider, model string) error {
	_, err := s.db.Exec(
		`insert into sessions (id, title, provider, model, updated_at) values (?, ?, ?, ?, current_timestamp)`,
		id, title, provider, model,
	)
	return err
}

func (s *Store) TouchSession(id, title string) error {
	_, err := s.db.Exec(`update sessions set title = ?, updated_at = current_timestamp where id = ?`, title, id)
	return err
}

func (s *Store) AddSessionTokens(id string, inputTokens, outputTokens int) error {
	_, err := s.db.Exec(
		`update sessions
		 set input_tokens = input_tokens + ?,
		     output_tokens = output_tokens + ?,
		     updated_at = current_timestamp
		 where id = ?`,
		inputTokens, outputTokens, id,
	)
	return err
}

func (s *Store) LatestSession() (Session, bool, error) {
	rows, err := s.db.Query(`select id, title, provider, model, input_tokens, output_tokens, created_at, updated_at from sessions order by updated_at desc limit 1`)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Session{}, false, rows.Err()
	}
	sess, err := scanSession(rows)
	return sess, err == nil, err
}

func (s *Store) ListSessions(limit int) ([]Session, error) {
	rows, err := s.db.Query(`select id, title, provider, model, input_tokens, output_tokens, created_at, updated_at from sessions order by updated_at desc limit ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) DeleteSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`delete from messages where session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`delete from workspace_saves where session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`delete from sessions where id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AddMessage(sessionID, role, content string) error {
	return s.AddMessageWithTools(sessionID, role, content, "", "")
}

func (s *Store) AddMessageWithTools(sessionID, role, content, toolCalls, toolCallID string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	blob, err := s.encrypt(content)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`insert into messages (session_id, role, content, tool_calls, tool_call_id) values (?, ?, ?, ?, ?)`,
		sessionID, role, blob, toolCalls, toolCallID,
	)
	return err
}

func (s *Store) Messages(sessionID string) ([]Message, error) {
	if !s.unlocked {
		return nil, errors.New("database is locked")
	}
	rows, err := s.db.Query(`select id, session_id, role, content, tool_calls, tool_call_id, created_at from messages where session_id = ? order by id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []Message
	for rows.Next() {
		var msg Message
		var enc string
		var toolCalls, toolCallID sql.NullString
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &enc, &toolCalls, &toolCallID, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msg.Content, err = s.decrypt(enc)
		if err != nil {
			return nil, err
		}
		if toolCalls.Valid {
			msg.ToolCalls = toolCalls.String
		}
		if toolCallID.Valid {
			msg.ToolCallID = toolCallID.String
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *Store) SaveWorkspace(name, sessionID, snapshot string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	blob, err := s.encrypt(snapshot)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`insert into workspace_saves (name, session_id, snapshot) values (?, ?, ?)`, name, sessionID, blob)
	return err
}

func (s *Store) WorkspaceSaves(limit int) ([]WorkspaceSave, error) {
	rows, err := s.db.Query(`select id, name, session_id, created_at from workspace_saves order by created_at desc limit ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var saves []WorkspaceSave
	for rows.Next() {
		var save WorkspaceSave
		if err := rows.Scan(&save.ID, &save.Name, &save.SessionID, &save.CreatedAt); err != nil {
			return nil, err
		}
		saves = append(saves, save)
	}
	return saves, rows.Err()
}

func (s *Store) Remember(key, value, tags string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	blob, err := s.encrypt(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`insert into memories (key, value, tags, updated_at)
		 values (?, ?, ?, current_timestamp)
		 on conflict(key) do update set value = excluded.value, tags = excluded.tags, updated_at = current_timestamp`,
		key, blob, tags,
	)
	return err
}

func (s *Store) ForgetMemory(key string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	_, err := s.db.Exec(`delete from memories where key = ?`, key)
	return err
}

func (s *Store) Memories(limit int) ([]Memory, error) {
	if !s.unlocked {
		return nil, errors.New("database is locked")
	}
	rows, err := s.db.Query(`select key, value, coalesce(tags, ''), created_at, updated_at from memories order by updated_at desc limit ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanMemories(rows)
}

func (s *Store) RecallMemories(query string, limit int) ([]Memory, error) {
	if !s.unlocked {
		return nil, errors.New("database is locked")
	}
	rows, err := s.db.Query(`select key, value, coalesce(tags, ''), created_at, updated_at from memories order by updated_at desc limit 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memories, err := s.scanMemories(rows)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		if len(memories) > limit {
			return memories[:limit], nil
		}
		return memories, nil
	}
	filtered := make([]Memory, 0, min(limit, len(memories)))
	for _, memory := range memories {
		haystack := strings.ToLower(memory.Key + "\n" + memory.Tags + "\n" + memory.Value)
		if strings.Contains(haystack, query) {
			filtered = append(filtered, memory)
			if len(filtered) >= limit {
				break
			}
		}
	}
	return filtered, nil
}

func (s *Store) scanMemories(rows *sql.Rows) ([]Memory, error) {
	var memories []Memory
	for rows.Next() {
		var memory Memory
		var enc string
		if err := rows.Scan(&memory.Key, &enc, &memory.Tags, &memory.CreatedAt, &memory.UpdatedAt); err != nil {
			return nil, err
		}
		value, err := s.decrypt(enc)
		if err != nil {
			return nil, err
		}
		memory.Value = value
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}

func (s *Store) unlockWith(password string) {
	sum := sha3.Sum256([]byte("weazlchat/local-only/" + password))
	s.key = sum[:]
	s.unlocked = true
}

func (s *Store) encrypt(plain string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := append(nonce, gcm.Seal(nil, nonce, []byte(plain), nil)...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (s *Store) decrypt(blob string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("encrypted payload is too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}
	return string(plain), nil
}

func scanSession(rows interface {
	Scan(dest ...any) error
}) (Session, error) {
	var sess Session
	return sess, rows.Scan(&sess.ID, &sess.Title, &sess.Provider, &sess.Model, &sess.InputTokens, &sess.OutputTokens, &sess.CreatedAt, &sess.UpdatedAt)
}

func (s *Store) ensureColumn(table, name, spec string) error {
	rows, err := s.db.Query(`pragma table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`alter table ` + table + ` add column ` + name + ` ` + spec)
	return err
}

func mkdirFor(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}

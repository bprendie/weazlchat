package storage

import "time"

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
	ID               int64
	Name             string
	SessionID        string
	ThroughMessageID int64
	CreatedAt        time.Time
}

type Memory struct {
	Key       string
	Value     string
	Tags      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ContextCheckpoint struct {
	ID               int64
	SessionID        string
	ThroughMessageID int64
	Summary          string
	CreatedAt        time.Time
}

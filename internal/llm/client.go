package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
)

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type StreamEvent struct {
	Type      string     // "content", "tool_call", "done"
	Content   string     // Text content chunk
	ToolCalls []ToolCall // Tool calls from assistant
	Usage     Usage      // Token usage (only on done)
	Error     error      // Error if any
}

type Client struct {
	provider config.Provider
	http     *http.Client
	tools    []map[string]any
}

func New(provider config.Provider) Client {
	return Client{
		provider: provider,
		http:     &http.Client{Timeout: 0},
	}
}

// WithTools adds tool definitions to the client
func (c Client) WithTools(tools []map[string]any) Client {
	c.tools = tools
	return c
}

func (c Client) Stream(ctx context.Context, history []storage.Message, prompt string, onEvent func(StreamEvent)) (Usage, error) {
	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		messages := chatMessages(history, prompt)
		return c.streamOpenAICompat(ctx, messages, onEvent)
	case "ollama":
		messages := ollamaChatMessages(history, prompt)
		return c.streamOllama(ctx, messages, onEvent)
	default:
		return Usage{}, fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

func (c Client) Summarize(ctx context.Context, transcript string, targetTokens int) (string, error) {
	if targetTokens <= 0 {
		targetTokens = 500
	}
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "You summarize conversation history for future context. Preserve user goals, decisions, durable facts, tool results, file paths, commands, unresolved tasks, and important constraints. Drop routine back-and-forth, duplicated text, and stale details. Be concise and do not invent details.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Create a compact checkpoint summary of this conversation in about %d tokens. This summary will replace the earlier messages in context. Use short sections for current objective, decisions/constraints, important files or tool results, and open next steps when applicable.\n\n%s", targetTokens, transcript),
		},
	}
	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		return c.completeOpenAICompat(ctx, messages, targetTokens+200)
	case "ollama":
		return c.completeOllama(ctx, messages, targetTokens+200)
	default:
		return "", fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

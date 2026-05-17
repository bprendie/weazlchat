package llm

import (
	"context"
	"fmt"
	"strings"
)

const summarySystemPrompt = "You summarize conversation history for future context. Preserve user goals, decisions, durable facts, tool results, file paths, commands, unresolved tasks, and important constraints. Drop routine back-and-forth, duplicated text, and stale details. Be concise and do not invent details."

func (c Client) Summarize(ctx context.Context, transcript string, targetTokens int) (string, error) {
	if targetTokens <= 0 {
		targetTokens = 500
	}
	messages := summaryMessages(transcript, targetTokens)
	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		return c.completeOpenAICompat(ctx, messages, targetTokens+200)
	case "ollama":
		return c.completeOllama(ctx, messages, targetTokens+200)
	default:
		return "", fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

func summaryMessages(transcript string, targetTokens int) []ChatMessage {
	return []ChatMessage{
		{
			Role:    "system",
			Content: summarySystemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Create a compact checkpoint summary of this conversation in about %d tokens. This summary will replace the earlier messages in context. Use short sections for current objective, decisions/constraints, important files or tool results, and open next steps when applicable.\n\n%s", targetTokens, transcript),
		},
	}
}

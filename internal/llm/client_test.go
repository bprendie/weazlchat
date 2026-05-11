package llm

import (
	"strings"
	"testing"

	"github.com/bprendie/weazlchat/internal/storage"
)

func TestChatMessagesDoesNotAppendEmptyPrompt(t *testing.T) {
	history := []storage.Message{
		{Role: "assistant", ToolCalls: `[{"id":"call_1","type":"function","function":{"name":"calculate","arguments":"{\"operation\":\"add\",\"a\":1,\"b\":2}"}}]`},
		{Role: "tool", Content: "1 + 2 = 3", ToolCallID: "call_1"},
	}

	messages := chatMessages(history, "")

	if len(messages) != 4 {
		t.Fatalf("message count = %d, want 4: %#v", len(messages), messages)
	}
	if messages[0].Role != "system" || messages[0].Content != markdownResponseSystemPrompt {
		t.Fatalf("system markdown prompt missing: %#v", messages[0])
	}
	if messages[1].Role != "system" || !strings.Contains(messages[1].Content, "Current local date/time") {
		t.Fatalf("current date system prompt missing: %#v", messages[1])
	}
	if messages[2].Role != "assistant" || len(messages[2].ToolCalls) != 1 {
		t.Fatalf("assistant tool calls were not preserved: %#v", messages[2])
	}
	if messages[2].Content != "" {
		t.Fatalf("assistant tool-call content = %q, want empty", messages[2].Content)
	}
	if messages[3].Role != "tool" || messages[3].ToolCallID != "call_1" {
		t.Fatalf("tool result metadata was not preserved: %#v", messages[3])
	}
}

func TestChatMessagesAppendsNonEmptyPrompt(t *testing.T) {
	messages := chatMessages(nil, "hello")

	if len(messages) != 3 {
		t.Fatalf("message count = %d, want 3", len(messages))
	}
	if messages[0].Role != "system" {
		t.Fatalf("first message = %#v, want system", messages[0])
	}
	if messages[1].Role != "system" || !strings.Contains(messages[1].Content, "Current local date/time") {
		t.Fatalf("current date system prompt missing: %#v", messages[1])
	}
	if messages[2].Role != "user" || messages[2].Content != "hello" {
		t.Fatalf("message = %#v, want user hello", messages[2])
	}
}

func TestOllamaChatMessagesUseToolNameAndObjectArguments(t *testing.T) {
	history := []storage.Message{
		{Role: "assistant", ToolCalls: `[{"id":"call_1","type":"function","function":{"name":"calculate","arguments":"{\"operation\":\"add\",\"a\":1,\"b\":2}"}}]`},
		{Role: "tool", Content: "1 + 2 = 3", ToolCallID: "call_1"},
	}

	messages := ollamaChatMessages(history, "")

	if len(messages) != 4 {
		t.Fatalf("message count = %d, want 4: %#v", len(messages), messages)
	}
	if messages[0].Role != "system" || messages[0].Content != markdownResponseSystemPrompt {
		t.Fatalf("system markdown prompt missing: %#v", messages[0])
	}
	if messages[1].Role != "system" || !strings.Contains(messages[1].Content, "Current local date/time") {
		t.Fatalf("current date system prompt missing: %#v", messages[1])
	}
	if messages[2].Role != "assistant" || len(messages[2].ToolCalls) != 1 {
		t.Fatalf("assistant tool calls were not converted: %#v", messages[2])
	}
	call := messages[2].ToolCalls[0]
	if call.Function.Name != "calculate" {
		t.Fatalf("tool name = %q, want calculate", call.Function.Name)
	}
	if call.Function.Arguments["operation"] != "add" || call.Function.Arguments["a"] != float64(1) {
		t.Fatalf("tool arguments were not decoded as an object: %#v", call.Function.Arguments)
	}
	if messages[3].Role != "tool" || messages[3].ToolName != "calculate" || messages[3].Content != "1 + 2 = 3" {
		t.Fatalf("tool result was not converted for Ollama: %#v", messages[3])
	}
}

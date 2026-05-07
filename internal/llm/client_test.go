package llm

import (
	"testing"

	"github.com/bprendie/weazlchat/internal/storage"
)

func TestChatMessagesDoesNotAppendEmptyPrompt(t *testing.T) {
	history := []storage.Message{
		{Role: "assistant", ToolCalls: `[{"id":"call_1","type":"function","function":{"name":"calculate","arguments":"{\"operation\":\"add\",\"a\":1,\"b\":2}"}}]`},
		{Role: "tool", Content: "1 + 2 = 3", ToolCallID: "call_1"},
	}

	messages := chatMessages(history, "")

	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2: %#v", len(messages), messages)
	}
	if messages[0].Role != "assistant" || len(messages[0].ToolCalls) != 1 {
		t.Fatalf("assistant tool calls were not preserved: %#v", messages[0])
	}
	if messages[0].Content != "" {
		t.Fatalf("assistant tool-call content = %q, want empty", messages[0].Content)
	}
	if messages[1].Role != "tool" || messages[1].ToolCallID != "call_1" {
		t.Fatalf("tool result metadata was not preserved: %#v", messages[1])
	}
}

func TestChatMessagesAppendsNonEmptyPrompt(t *testing.T) {
	messages := chatMessages(nil, "hello")

	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hello" {
		t.Fatalf("message = %#v, want user hello", messages[0])
	}
}

func TestOllamaChatMessagesUseToolNameAndObjectArguments(t *testing.T) {
	history := []storage.Message{
		{Role: "assistant", ToolCalls: `[{"id":"call_1","type":"function","function":{"name":"calculate","arguments":"{\"operation\":\"add\",\"a\":1,\"b\":2}"}}]`},
		{Role: "tool", Content: "1 + 2 = 3", ToolCallID: "call_1"},
	}

	messages := ollamaChatMessages(history, "")

	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2: %#v", len(messages), messages)
	}
	if messages[0].Role != "assistant" || len(messages[0].ToolCalls) != 1 {
		t.Fatalf("assistant tool calls were not converted: %#v", messages[0])
	}
	call := messages[0].ToolCalls[0]
	if call.Function.Name != "calculate" {
		t.Fatalf("tool name = %q, want calculate", call.Function.Name)
	}
	if call.Function.Arguments["operation"] != "add" || call.Function.Arguments["a"] != float64(1) {
		t.Fatalf("tool arguments were not decoded as an object: %#v", call.Function.Arguments)
	}
	if messages[1].Role != "tool" || messages[1].ToolName != "calculate" || messages[1].Content != "1 + 2 = 3" {
		t.Fatalf("tool result was not converted for Ollama: %#v", messages[1])
	}
}

package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bprendie/weazlchat/internal/storage"
)

const markdownResponseSystemPrompt = "Format normal responses as Markdown so headings, lists, code blocks, quotes, links, and tables render cleanly in the terminal. If the user explicitly requests a different raw format such as JSON, Python, SQL, CSV, or plain text, honor that requested format exactly."

func chatMessages(history []storage.Message, prompt string) []ChatMessage {
	messages := make([]ChatMessage, 0, len(history)+2)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt()})
	for _, msg := range history {
		cm := ChatMessage{Role: msg.Role, Content: msg.Content}
		if msg.Role == "assistant" && msg.ToolCalls != "" {
			var toolCalls []ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err == nil {
				cm.ToolCalls = toolCalls
				cm.Content = ""
			}
		}
		if msg.Role == "tool" {
			cm.ToolCallID = msg.ToolCallID
		}
		messages = append(messages, cm)
	}
	if strings.TrimSpace(prompt) != "" {
		messages = append(messages, ChatMessage{Role: "user", Content: prompt})
	}
	return messages
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
}

type ollamaToolCall struct {
	Type     string `json:"type,omitempty"`
	Function struct {
		Index     int            `json:"index,omitempty"`
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

func ollamaChatMessages(history []storage.Message, prompt string) []ollamaMessage {
	messages := make([]ollamaMessage, 0, len(history)+2)
	messages = append(messages, ollamaMessage{Role: "system", Content: systemPrompt()})
	toolNames := make(map[string]string)
	for _, msg := range history {
		cm := ollamaMessage{Role: msg.Role, Content: msg.Content}
		if msg.Role == "assistant" && msg.ToolCalls != "" {
			var toolCalls []ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err == nil {
				cm.Content = ""
				cm.ToolCalls = make([]ollamaToolCall, 0, len(toolCalls))
				for i, call := range toolCalls {
					toolNames[call.ID] = call.Function.Name
					var args map[string]any
					if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
						args = map[string]any{}
					}
					var tc ollamaToolCall
					tc.Type = "function"
					tc.Function.Index = i
					tc.Function.Name = call.Function.Name
					tc.Function.Arguments = args
					cm.ToolCalls = append(cm.ToolCalls, tc)
				}
			}
		}
		if msg.Role == "tool" {
			cm.ToolName = toolNames[msg.ToolCallID]
			if cm.ToolName == "" {
				cm.ToolName = msg.ToolCallID
			}
		}
		messages = append(messages, cm)
	}
	if strings.TrimSpace(prompt) != "" {
		messages = append(messages, ollamaMessage{Role: "user", Content: prompt})
	}
	return messages
}

func systemPrompt() string {
	return markdownResponseSystemPrompt + "\n\n" + currentDateSystemPrompt()
}

func currentDateSystemPrompt() string {
	now := time.Now()
	location := time.Local.String()
	if location == "" {
		location = "local"
	}
	return fmt.Sprintf("Current local date/time for this request: %s (%s). Treat words like today, tomorrow, yesterday, latest, and current relative to this timestamp unless tool output says otherwise.", now.Format(time.RFC1123Z), location)
}

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func (c Client) streamOllama(ctx context.Context, messages []ollamaMessage, onEvent func(StreamEvent)) (Usage, error) {
	reqBody := map[string]any{
		"model":    c.provider.Model,
		"messages": messages,
		"stream":   true,
	}

	if len(c.tools) > 0 {
		reqBody["tools"] = c.tools
	}

	resp, err := c.post(ctx, "/api/chat", reqBody)
	if err != nil {
		return Usage{}, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var usage Usage

	for {
		var chunk ollamaStreamChunk
		if err := dec.Decode(&chunk); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return usage, err
		}
		if chunk.Error != "" {
			return usage, errors.New(chunk.Error)
		}
		if chunk.Message.Content != "" {
			onEvent(StreamEvent{Type: "content", Content: chunk.Message.Content})
		}
		if len(chunk.Message.ToolCalls) > 0 {
			onEvent(StreamEvent{Type: "tool_call", ToolCalls: ollamaToolCalls(chunk)})
		}
		if chunk.Done {
			usage.InputTokens = chunk.PromptEvalCount
			usage.OutputTokens = chunk.EvalCount
			break
		}
	}
	return usage, nil
}

type ollamaStreamChunk struct {
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Function struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"message"`
	Done            bool   `json:"done"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	Error           string `json:"error"`
}

func ollamaToolCalls(chunk ollamaStreamChunk) []ToolCall {
	toolCalls := make([]ToolCall, len(chunk.Message.ToolCalls))
	for i, tc := range chunk.Message.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		toolCalls[i] = ToolCall{
			ID:   fmt.Sprintf("call_%d", i),
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: string(argsJSON),
			},
		}
	}
	return toolCalls
}

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Client struct {
	provider config.Provider
	http     *http.Client
}

func New(provider config.Provider) Client {
	return Client{
		provider: provider,
		http:     &http.Client{Timeout: 0},
	}
}

func (c Client) Stream(ctx context.Context, history []storage.Message, prompt string, onChunk func(string)) (Usage, error) {
	messages := make([]ChatMessage, 0, len(history)+1)
	for _, msg := range history {
		messages = append(messages, ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: prompt})

	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		return c.streamOpenAICompat(ctx, messages, onChunk)
	case "ollama":
		return c.streamOllama(ctx, messages, onChunk)
	default:
		return Usage{}, fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

func (c Client) streamOpenAICompat(ctx context.Context, messages []ChatMessage, onChunk func(string)) (Usage, error) {
	reqBody := map[string]any{
		"model":       c.provider.Model,
		"messages":    messages,
		"temperature": 0.7,
		"stream":      true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
	resp, err := c.post(ctx, "/v1/chat/completions", reqBody)
	if err != nil {
		return Usage{}, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var usage Usage
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return usage, err
		}
		if chunk.Error != nil {
			return usage, errors.New(chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				onChunk(choice.Delta.Content)
			}
		}
	}
	return usage, scanner.Err()
}

func (c Client) streamOllama(ctx context.Context, messages []ChatMessage, onChunk func(string)) (Usage, error) {
	reqBody := map[string]any{
		"model":    c.provider.Model,
		"messages": messages,
		"stream":   true,
	}
	resp, err := c.post(ctx, "/api/chat", reqBody)
	if err != nil {
		return Usage{}, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var usage Usage
	for {
		var chunk struct {
			Message         ChatMessage `json:"message"`
			Done            bool        `json:"done"`
			PromptEvalCount int         `json:"prompt_eval_count"`
			EvalCount       int         `json:"eval_count"`
			Error           string      `json:"error"`
		}
		if err := dec.Decode(&chunk); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return usage, err
		}
		if chunk.Error != "" {
			return usage, errors.New(chunk.Error)
		}
		if chunk.Message.Content != "" {
			onChunk(chunk.Message.Content)
		}
		if chunk.Done {
			usage.InputTokens = chunk.PromptEvalCount
			usage.OutputTokens = chunk.EvalCount
			break
		}
	}
	return usage, nil
}

func (c Client) post(ctx context.Context, path string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := baseURL(c.provider) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return resp, nil
}

func baseURL(provider config.Provider) string {
	u := strings.TrimRight(strings.TrimSpace(provider.ServerURL), "/")
	switch strings.ToLower(provider.Type) {
	case "vllm":
		u = strings.TrimSuffix(u, "/v1")
	case "ollama":
		u = strings.TrimSuffix(u, "/api")
	}
	return u
}

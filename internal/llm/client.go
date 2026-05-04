package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	provider config.Provider
	http     *http.Client
}

func New(provider config.Provider) Client {
	return Client{
		provider: provider,
		http:     &http.Client{Timeout: 5 * time.Minute},
	}
}

func (c Client) Complete(ctx context.Context, history []storage.Message, prompt string) (string, error) {
	messages := make([]ChatMessage, 0, len(history)+1)
	for _, msg := range history {
		messages = append(messages, ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: prompt})

	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		return c.openAICompat(ctx, messages)
	case "ollama":
		return c.ollama(ctx, messages)
	default:
		return "", fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

func (c Client) openAICompat(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := map[string]any{
		"model":       c.provider.Model,
		"messages":    messages,
		"temperature": 0.7,
	}
	var respBody struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := c.postJSON(ctx, "/v1/chat/completions", reqBody, &respBody); err != nil {
		return "", err
	}
	if respBody.Error != nil {
		return "", errors.New(respBody.Error.Message)
	}
	if len(respBody.Choices) == 0 {
		return "", errors.New("provider returned no choices")
	}
	return strings.TrimSpace(respBody.Choices[0].Message.Content), nil
}

func (c Client) ollama(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := map[string]any{
		"model":    c.provider.Model,
		"messages": messages,
		"stream":   false,
	}
	var respBody struct {
		Message ChatMessage `json:"message"`
		Error   string      `json:"error"`
	}
	if err := c.postJSON(ctx, "/api/chat", reqBody, &respBody); err != nil {
		return "", err
	}
	if respBody.Error != "" {
		return "", errors.New(respBody.Error)
	}
	return strings.TrimSpace(respBody.Message.Content), nil
}

func (c Client) postJSON(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := baseURL(c.provider) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
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

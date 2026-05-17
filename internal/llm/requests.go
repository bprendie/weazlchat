package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

func (c Client) completeOpenAICompat(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	reqBody := map[string]any{
		"model":       c.provider.Model,
		"messages":    messages,
		"temperature": 0.2,
		"stream":      false,
		"max_tokens":  maxTokens,
	}
	resp, err := c.post(ctx, "/v1/chat/completions", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Error != nil {
		return "", errors.New(body.Error.Message)
	}
	if len(body.Choices) == 0 {
		return "", errors.New("empty completion response")
	}
	return strings.TrimSpace(body.Choices[0].Message.Content), nil
}

func (c Client) completeOllama(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	reqBody := map[string]any{
		"model":    c.provider.Model,
		"messages": messages,
		"stream":   false,
		"options": map[string]any{
			"num_predict": maxTokens,
			"temperature": 0.2,
		},
	}
	resp, err := c.post(ctx, "/api/chat", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Error != "" {
		return "", errors.New(body.Error)
	}
	return strings.TrimSpace(body.Message.Content), nil
}

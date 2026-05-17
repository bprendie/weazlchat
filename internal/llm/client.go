package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
)

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

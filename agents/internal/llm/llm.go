// Package llm is a small Completer. Text defaults to DeepSeek. Receipt photos use Gemini. No local models.
package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderDeepSeek  Provider = "deepseek"
)

type Image struct {
	MIME string
	Data []byte
}

type Config struct {
	Provider Provider
	Model    string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
}

type Message struct {
	Role    string
	Content string
}

type Request struct {
	System   string
	Messages []Message
	Images   []Image
}

type Completer interface {
	Complete(ctx context.Context, req Request) (string, error)
}

func New(cfg Config) (Completer, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 120 * time.Second
	}
	if cfg.APIKey == "" {
		return nil, errors.New("LLM_API_KEY is empty")
	}
	if cfg.Model == "" {
		return nil, errors.New("LLM_MODEL is empty")
	}
	httpClient := &http.Client{Timeout: cfg.Timeout}
	switch cfg.Provider {
	case ProviderOpenAI, "":
		base := firstNonEmpty(cfg.BaseURL, defaultOpenAIBase)
		return &openAICompat{client: httpClient, apiKey: cfg.APIKey, model: cfg.Model, baseURL: strings.TrimRight(base, "/")}, nil
	case ProviderDeepSeek:
		base := firstNonEmpty(cfg.BaseURL, defaultDeepSeekBase)
		return &openAICompat{client: httpClient, apiKey: cfg.APIKey, model: cfg.Model, baseURL: strings.TrimRight(base, "/")}, nil
	case ProviderGemini:
		base := firstNonEmpty(cfg.BaseURL, defaultGeminiBase)
		return &openAICompat{client: httpClient, apiKey: cfg.APIKey, model: cfg.Model, baseURL: strings.TrimRight(base, "/")}, nil
	case ProviderAnthropic:
		base := firstNonEmpty(cfg.BaseURL, defaultAnthropicBase)
		return &anthropicClient{client: httpClient, apiKey: cfg.APIKey, model: cfg.Model, baseURL: strings.TrimRight(base, "/")}, nil
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (openai|anthropic|deepseek|gemini)", cfg.Provider)
	}
}

type router struct {
	text, vision Completer
}

func (r *router) Complete(ctx context.Context, req Request) (string, error) {
	if len(req.Images) > 0 {
		if r.vision == nil {
			return "", errors.New("photo OCR needs GEMINI_API_KEY")
		}
		return r.vision.Complete(ctx, req)
	}
	return r.text.Complete(ctx, req)
}

func ParseProvider(s string) Provider {
	return Provider(strings.ToLower(strings.TrimSpace(s)))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type openAICompat struct {
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	Temperature    float64             `json:"temperature"`
	ResponseFormat map[string]string   `json:"response_format"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openAIChatResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *openAICompat) Complete(ctx context.Context, req Request) (string, error) {
	msgs := make([]openAIChatMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, openAIChatMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, openAIChatMessage{Role: m.Role, Content: m.Content})
	}
	if len(req.Images) > 0 {
		msgs = attachImages(msgs, req.Images)
	}
	body, err := json.Marshal(openAIChatRequest{
		Model:          c.model,
		Messages:       msgs,
		Temperature:    0.2,
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	var parsed openAIChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("openai: decode: %w (%s)", err, truncate(raw, 300))
	}
	if parsed.Error.Message != "" {
		return "", fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, truncate(raw, 400))
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("openai: empty choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

type anthropicClient struct {
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (c *anthropicClient) Complete(ctx context.Context, req Request) (string, error) {
	msgs := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := m.Role
		if role == "system" {
			continue
		}
		if role != "user" && role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
	}
	if len(msgs) == 0 {
		return "", errors.New("anthropic: no user messages")
	}
	body, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    req.System,
		Messages:  msgs,
	})
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: decode: %w (%s)", err, truncate(raw, 300))
	}
	if parsed.Error.Message != "" {
		return "", fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, truncate(raw, 400))
	}
	var b strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	if b.Len() == 0 {
		return "", errors.New("anthropic: empty content")
	}
	return b.String(), nil
}

func attachImages(msgs []openAIChatMessage, images []Image) []openAIChatMessage {
	if len(msgs) == 0 {
		msgs = append(msgs, openAIChatMessage{Role: "user", Content: ""})
	}
	idx := len(msgs) - 1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			idx = i
			break
		}
	}
	text := ""
	switch v := msgs[idx].Content.(type) {
	case string:
		text = v
	}
	if strings.TrimSpace(text) == "" {
		text = "Read the attached image. Do not invent figures that are not visible."
	}
	parts := []map[string]any{{"type": "text", "text": text}}
	for _, img := range images {
		if len(img.Data) == 0 {
			continue
		}
		mime := img.MIME
		if mime == "" {
			mime = "image/jpeg"
		}
		parts = append(parts, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}
	if len(parts) == 1 {
		return msgs
	}
	msgs[idx].Content = parts
	return msgs
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

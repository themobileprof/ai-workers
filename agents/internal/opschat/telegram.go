package opschat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultWebhookSecret matches the token Telegram already sends to /webhook/telegram-ops/webhook.
const DefaultWebhookSecret = "opsTelegram00001_d7e8f901-2345-4abc-9def-telegrtrig001"

const DefaultWebhookURL = "https://workers.themobileprof.com/webhook/telegram-ops/webhook"

// Bot talks to api.telegram.org.
type Bot struct {
	token  string
	secret string
	http   *http.Client
}

func NewBot(token, secret string) *Bot {
	token = strings.TrimSpace(token)
	secret = strings.TrimSpace(secret)
	if secret == "" {
		secret = DefaultWebhookSecret
	}
	if token == "" {
		return nil
	}
	return &Bot{
		token:  token,
		secret: secret,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (b *Bot) Secret() string {
	if b == nil {
		return ""
	}
	return b.secret
}

func (b *Bot) SetWebhook(ctx context.Context, url string) error {
	if b == nil {
		return fmt.Errorf("telegram bot is not configured")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		url = DefaultWebhookURL
	}
	body, _ := json.Marshal(map[string]any{
		"url":             url,
		"secret_token":    b.secret,
		"allowed_updates": []string{"message"},
	})
	return b.post(ctx, "setWebhook", body, nil)
}

func (b *Bot) Send(ctx context.Context, chatID, text string) error {
	if b == nil {
		return fmt.Errorf("telegram bot is not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    ClipText(text),
	})
	return b.post(ctx, "sendMessage", body, nil)
}

func (b *Bot) Download(ctx context.Context, fileID string) (b64 string, mime string, err error) {
	if b == nil {
		return "", "", fmt.Errorf("telegram bot is not configured")
	}
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return "", "", fmt.Errorf("empty file_id")
	}
	var got struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	q, _ := json.Marshal(map[string]string{"file_id": fileID})
	if err := b.post(ctx, "getFile", q, &got); err != nil {
		return "", "", err
	}
	if !got.OK || got.Result.FilePath == "" {
		return "", "", fmt.Errorf("telegram getFile failed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/file/bot"+b.token+"/"+got.Result.FilePath, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("telegram file HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		return "", "", err
	}
	mime = resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	return base64.StdEncoding.EncodeToString(raw), mime, nil
}

func (b *Bot) post(ctx context.Context, method string, payload []byte, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+b.token+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram %s HTTP %d: %s", method, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

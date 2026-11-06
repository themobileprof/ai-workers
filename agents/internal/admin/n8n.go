package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

const (
	n8nBooksPath  = "/webhook/books-write"
	n8nOutboxPath = "/webhook/email-outbox"
)

type n8nClient struct {
	base  string
	token string
	http  *http.Client
}

func newN8nClient(base, token string) *n8nClient {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	token = strings.TrimSpace(token)
	if base == "" {
		return nil
	}
	return &n8nClient{
		base:  base,
		token: token,
		http:  &http.Client{Timeout: 90 * time.Second},
	}
}

func n8nBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("N8N_BASE_URL")); v != "" {
		return v
	}
	return "http://n8n:5678"
}

func (s *Server) WithN8n(base, token string) {
	if s == nil {
		return
	}
	if strings.TrimSpace(base) == "" {
		base = n8nBaseURL()
	}
	s.n8n = newN8nClient(base, token)
}

func (s *Server) maybeBooksWrite(ctx context.Context, dept string, resp contract.Response) string {
	body, ok := booksWriteBody(dept, resp)
	if !ok {
		return ""
	}
	if s == nil || s.n8n == nil {
		return ""
	}
	out, err := s.n8n.post(ctx, n8nBooksPath, body)
	if err != nil {
		return "Zoho Books: " + err.Error()
	}
	return extractField(out, "audit_text")
}

func (s *Server) runOutbox(ctx context.Context, action, chatID string) string {
	if s == nil || s.n8n == nil {
		return "Email outbox is not reachable. Use Email on the desk."
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		action = "send"
	}
	out, err := s.n8n.post(ctx, n8nOutboxPath, map[string]any{
		"action":  action,
		"chat_id": chatID,
	})
	if err != nil {
		return "Email outbox: " + err.Error()
	}
	text := extractField(out, "text")
	if text == "" {
		text = extractField(out, "audit_text")
	}
	if text == "" {
		switch action {
		case "kill":
			return "Dropped the pending draft, or there was none."
		case "set_chat":
			return ""
		default:
			return "No pending email draft."
		}
	}
	return text
}

func (s *Server) rememberOpsChat(ctx context.Context, chatID string) {
	if s == nil || s.n8n == nil || strings.TrimSpace(chatID) == "" {
		return
	}
	_, _ = s.n8n.post(ctx, n8nOutboxPath, map[string]any{
		"action":  "set_chat",
		"chat_id": chatID,
	})
}

func booksWriteBody(dept string, resp contract.Response) (map[string]any, bool) {
	if resp.Status != contract.StatusSuccess {
		return nil, false
	}
	dept = strings.TrimSpace(strings.ToLower(dept))
	if dept != "accounts" && dept != "internal-ops" {
		return nil, false
	}
	sd := resp.StructuredData
	if sd == nil {
		return nil, false
	}
	if dept == "internal-ops" && strings.ToLower(strVal(sd["task_type"])) != "accounts" {
		return nil, false
	}
	action := strings.ToLower(strings.TrimSpace(strVal(sd["zoho_action"])))
	allowed := map[string]bool{
		"expense": true,
		"bill":    true,
		"invoice": true,
		"lookup":  true,
		"preview": true,
	}
	if !allowed[action] && boolVal(sd["record_expense"]) {
		action = "expense"
	}
	if !allowed[action] {
		return nil, false
	}
	out := map[string]any{}
	for k, v := range sd {
		out[k] = v
	}
	out["action"] = action
	out["structured_data"] = sd
	return out, true
}

func (c *n8nClient) post(ctx context.Context, path string, body any) (any, error) {
	if c == nil {
		return nil, fmt.Errorf("n8n is not configured")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = res.Status
		}
		return nil, fmt.Errorf("HTTP %d %s", res.StatusCode, clipN8nErr(msg))
	}
	return decodeN8nJSON(b), nil
}

func decodeN8nJSON(raw []byte) any {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return map[string]any{"text": string(raw)}
	}
	return unwrapN8n(v)
}

func unwrapN8n(v any) any {
	switch t := v.(type) {
	case []any:
		if len(t) == 0 {
			return nil
		}
		return unwrapN8n(t[0])
	case map[string]any:
		if inner, ok := t["json"]; ok && len(t) <= 3 {
			return unwrapN8n(inner)
		}
		return t
	default:
		return v
	}
}

func extractField(v any, keys ...string) string {
	v = unwrapN8n(v)
	m, ok := v.(map[string]any)
	if !ok {
		return strings.TrimSpace(strVal(v))
	}
	for _, k := range keys {
		if s := strings.TrimSpace(strVal(m[k])); s != "" {
			return s
		}
	}
	return ""
}

func clipN8nErr(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 240 {
		return s[:240]
	}
	return s
}

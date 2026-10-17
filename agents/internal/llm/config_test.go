package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigsFromEnvProviderSpecificKeys(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "deepseek")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("DEEPSEEK_API_KEY", "sk-deepseek-test")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-chat")
	t.Setenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
	t.Setenv("GEMINI_API_KEY", "AQ.test-gemini")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GOOGLE_API_KEY", "")

	text, vision := ConfigsFromEnv()
	if text.Provider != ProviderDeepSeek || text.APIKey != "sk-deepseek-test" || text.Model != "deepseek-chat" {
		t.Fatalf("text=%+v", text)
	}
	if vision.Provider != ProviderGemini || vision.APIKey != "AQ.test-gemini" || vision.Model != defaultGeminiModel {
		t.Fatalf("vision=%+v", vision)
	}
}

func TestConfigsFromEnvLegacyLLMLabels(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "deepseek")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_MODEL", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("LLM_API_KEY", "sk-legacy")
	t.Setenv("LLM_MODEL", "deepseek-chat")
	t.Setenv("LLM_BASE_URL", "https://api.deepseek.com")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	text, vision := ConfigsFromEnv()
	if text.APIKey != "sk-legacy" || text.Model != "deepseek-chat" {
		t.Fatalf("text=%+v", text)
	}
	if vision.APIKey != "" {
		t.Fatalf("unexpected vision %+v", vision)
	}
}

func TestAttachImagesBuildsDataURL(t *testing.T) {
	msgs := []openAIChatMessage{{Role: "user", Content: "read this"}}
	out := attachImages(msgs, []Image{{MIME: "image/png", Data: []byte{0x89, 0x50, 0x4E, 0x47}}})
	parts, ok := out[0].Content.([]map[string]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("content=%T %+v", out[0].Content, out[0].Content)
	}
	if parts[0]["text"] != "read this" {
		t.Fatalf("text part=%v", parts[0])
	}
	img, _ := parts[1]["image_url"].(map[string]string)
	if img["url"] != "data:image/png;base64,iVBORw==" {
		t.Fatalf("url=%s", img["url"])
	}
}

func TestNewRejectsEmptyKeyAndUnknownProvider(t *testing.T) {
	if _, err := New(Config{Provider: ProviderDeepSeek, Model: "deepseek-chat"}); err == nil {
		t.Fatal("empty key")
	}
	if _, err := New(Config{Provider: ProviderDeepSeek, APIKey: "sk"}); err == nil {
		t.Fatal("empty model")
	}
	if _, err := New(Config{Provider: "ollama", APIKey: "sk", Model: "x"}); err == nil {
		t.Fatal("unknown provider")
	}
	c, err := New(Config{Provider: ProviderDeepSeek, APIKey: "sk", Model: "deepseek-chat"})
	if err != nil || c == nil {
		t.Fatal(err)
	}
}

func TestNewStack(t *testing.T) {
	clearLLMEnv(t)
	if _, err := NewStack(Config{}, Config{}); err == nil {
		t.Fatal("empty text")
	}
	s, err := NewStack(Config{Provider: ProviderDeepSeek, APIKey: "sk", Model: "deepseek-chat"}, Config{})
	if err != nil || s.VisionModel != "" {
		t.Fatalf("%+v %v", s, err)
	}
	s, err = NewStack(
		Config{Provider: ProviderDeepSeek, APIKey: "sk", Model: "deepseek-chat"},
		Config{Provider: ProviderGemini, APIKey: "g", Model: defaultGeminiModel},
	)
	if err != nil || s.VisionModel != defaultGeminiModel {
		t.Fatalf("vision %+v %v", s, err)
	}
}

func TestNewStackFallsBackWhenPrimaryHasNoKey(t *testing.T) {
	clearLLMEnv(t)
	t.Setenv("GEMINI_API_KEY", "AQ.fallback")
	t.Setenv("GEMINI_MODEL", defaultGeminiModel)
	s, err := NewStack(Config{Provider: ProviderDeepSeek}, Config{})
	if err != nil || s.Provider != ProviderGemini || s.Completer == nil {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestExtraTextConfigsSkipsPrimary(t *testing.T) {
	clearLLMEnv(t)
	t.Setenv("DEEPSEEK_API_KEY", "sk-ds")
	t.Setenv("GEMINI_API_KEY", "AQ.g")
	t.Setenv("OPENAI_API_KEY", "sk-oa")
	got := extraTextConfigs(Config{Provider: ProviderDeepSeek, APIKey: "sk-ds", Model: "deepseek-chat"})
	if len(got) != 2 || got[0].Provider != ProviderGemini || got[1].Provider != ProviderOpenAI {
		t.Fatalf("%+v", got)
	}
}

func TestFallbackCompleter(t *testing.T) {
	f := &fallback{chain: []Completer{
		stubLLM{err: errors.New("primary down")},
		stubLLM{text: "from gemini"},
	}}
	got, err := f.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil || got != "from gemini" {
		t.Fatalf("%s %v", got, err)
	}
	dead := &fallback{chain: []Completer{stubLLM{err: errors.New("a")}, stubLLM{err: errors.New("b")}}}
	if _, err := dead.Complete(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "a") {
		t.Fatalf("%v", err)
	}
}

func TestOpenAICompatPlainOmitsJSONMode(t *testing.T) {
	var sawFormat bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sawFormat = strings.Contains(string(raw), "response_format")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &openAICompat{client: srv.Client(), apiKey: "sk", model: "gpt", baseURL: srv.URL}
	if _, err := c.Complete(context.Background(), Request{Plain: true, Messages: []Message{{Role: "user", Content: "hi"}}}); err != nil {
		t.Fatal(err)
	}
	if sawFormat {
		t.Fatal("plain requests must omit response_format")
	}
}

func clearLLMEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"LLM_PROVIDER", "LLM_API_KEY", "LLM_MODEL", "LLM_BASE_URL",
		"DEEPSEEK_API_KEY", "DEEPSEEK_MODEL", "DEEPSEEK_BASE_URL",
		"GEMINI_API_KEY", "GEMINI_MODEL", "GEMINI_BASE_URL", "GOOGLE_API_KEY",
		"OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_BASE_URL",
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "ANTHROPIC_BASE_URL",
	} {
		t.Setenv(k, "")
	}
}

func TestRouter(t *testing.T) {
	r := &router{text: stubLLM{text: "text"}, vision: stubLLM{text: "vision"}}
	got, err := r.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil || got != "text" {
		t.Fatalf("%s %v", got, err)
	}
	got, err = r.Complete(context.Background(), Request{Images: []Image{{MIME: "image/jpeg", Data: []byte{1}}}})
	if err != nil || got != "vision" {
		t.Fatalf("vision %s %v", got, err)
	}
	bare := &router{text: stubLLM{text: "text"}}
	if _, err := bare.Complete(context.Background(), Request{Images: []Image{{Data: []byte{1}}}}); err == nil {
		t.Fatal("photos need vision")
	}
}

func TestOpenAICompatComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer sk" {
			http.Error(w, "nope", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &openAICompat{client: srv.Client(), apiKey: "sk", model: "deepseek-chat", baseURL: srv.URL}
	got, err := c.Complete(context.Background(), Request{System: "sys", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil || got != `{"ok":true}` {
		t.Fatalf("%s %v", got, err)
	}

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"message":"quota"}}`))
	}))
	t.Cleanup(errSrv.Close)
	c.client, c.baseURL = errSrv.Client(), errSrv.URL
	if _, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hi"}}}); err == nil || !strings.Contains(err.Error(), "quota") {
		t.Fatalf("%v", err)
	}
}

func TestAnthropicComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "sk" {
			http.Error(w, "nope", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hello"}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &anthropicClient{client: srv.Client(), apiKey: "sk", model: "claude", baseURL: srv.URL}
	got, err := c.Complete(context.Background(), Request{System: "sys", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil || got != "hello" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := c.Complete(context.Background(), Request{}); err == nil {
		t.Fatal("no user messages")
	}
}

func TestTruncate(t *testing.T) {
	if truncate([]byte("  hi  "), 10) != "hi" {
		t.Fatal("short")
	}
	if got := truncate([]byte("abcdef"), 3); got != "abc…" {
		t.Fatal(got)
	}
}

func TestEnvPrefixAndDefaultModel(t *testing.T) {
	if envPrefix(ProviderGemini) != "GEMINI" || envPrefix(ProviderAnthropic) != "ANTHROPIC" || envPrefix("") != "OPENAI" {
		t.Fatal("prefix")
	}
	if defaultModel(ProviderGemini) != defaultGeminiModel || defaultModel("") != defaultOpenAIModel {
		t.Fatal("model")
	}
}

type stubLLM struct {
	text string
	err  error
}

func (s stubLLM) Complete(context.Context, Request) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.text, nil
}

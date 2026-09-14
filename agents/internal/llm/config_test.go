package llm

import "testing"

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

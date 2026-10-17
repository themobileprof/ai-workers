package llm

import (
	"errors"
	"os"
	"strings"
)

const (
	ProviderGemini Provider = "gemini"

	defaultGeminiModel    = "gemini-3.6-flash"
	defaultDeepSeekModel  = "deepseek-chat"
	defaultOpenAIModel    = "gpt-4.1-mini"
	defaultAnthropicModel = "claude-sonnet-4-5"

	defaultGeminiBase    = "https://generativelanguage.googleapis.com/v1beta/openai"
	defaultDeepSeekBase  = "https://api.deepseek.com/v1"
	defaultOpenAIBase    = "https://api.openai.com/v1"
	defaultAnthropicBase = "https://api.anthropic.com"
)

// Stack is the text completer plus an optional Gemini vision client for photos.
type Stack struct {
	Completer   Completer
	Provider    Provider
	Model       string
	VisionModel string
}

func ConfigsFromEnv() (text Config, vision Config) {
	p := ParseProvider(os.Getenv("LLM_PROVIDER"))
	if p == "" {
		p = ProviderOpenAI
	}
	text = Config{
		Provider: p,
		Model:    firstNonEmpty(os.Getenv(envPrefix(p)+"_MODEL"), os.Getenv("LLM_MODEL"), defaultModel(p)),
		APIKey:   firstNonEmpty(os.Getenv(envPrefix(p)+"_API_KEY"), os.Getenv("LLM_API_KEY")),
		BaseURL:  firstNonEmpty(os.Getenv(envPrefix(p)+"_BASE_URL"), os.Getenv("LLM_BASE_URL")),
	}
	geminiKey := firstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY"))
	if p != ProviderGemini && geminiKey != "" {
		vision = Config{
			Provider: ProviderGemini,
			Model:    firstNonEmpty(os.Getenv("GEMINI_MODEL"), defaultGeminiModel),
			APIKey:   geminiKey,
			BaseURL:  firstNonEmpty(os.Getenv("GEMINI_BASE_URL"), defaultGeminiBase),
		}
	}
	if p == ProviderGemini && text.APIKey == "" {
		text.APIKey = geminiKey
	}
	return text, vision
}

func NewStack(text, vision Config) (Stack, error) {
	var chain []Completer
	var first Config
	var lastErr error
	try := func(cfg Config) {
		if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
			if lastErr == nil {
				lastErr = errors.New("LLM_API_KEY is empty")
			}
			return
		}
		if cfg.Timeout <= 0 && text.Timeout > 0 {
			cfg.Timeout = text.Timeout
		}
		c, err := New(cfg)
		if err != nil {
			lastErr = err
			return
		}
		if first.APIKey == "" {
			first = cfg
		}
		chain = append(chain, c)
	}
	try(text)
	for _, extra := range extraTextConfigs(text) {
		try(extra)
	}
	if len(chain) == 0 {
		if lastErr == nil {
			lastErr = errors.New("LLM_API_KEY is empty")
		}
		return Stack{}, lastErr
	}
	primary := Completer(chain[0])
	if len(chain) > 1 {
		primary = &fallback{chain: chain}
	}
	s := Stack{Completer: primary, Provider: first.Provider, Model: first.Model}
	if first.Provider == ProviderGemini {
		s.VisionModel = first.Model
		return s, nil
	}
	if vision.APIKey == "" {
		return s, nil
	}
	v, err := New(vision)
	if err != nil {
		return s, err
	}
	s.VisionModel = vision.Model
	s.Completer = &router{text: primary, vision: v}
	return s, nil
}

// extraTextConfigs are other providers with keys in the environment.
// Tried in order after the primary when that call errors (desk guide and workers).
func extraTextConfigs(primary Config) []Config {
	var out []Config
	for _, p := range []Provider{ProviderDeepSeek, ProviderGemini, ProviderOpenAI, ProviderAnthropic} {
		if p == primary.Provider {
			continue
		}
		key := strings.TrimSpace(os.Getenv(envPrefix(p) + "_API_KEY"))
		if p == ProviderGemini && key == "" {
			key = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
		}
		if key == "" {
			continue
		}
		out = append(out, Config{
			Provider: p,
			Model:    firstNonEmpty(os.Getenv(envPrefix(p)+"_MODEL"), defaultModel(p)),
			APIKey:   key,
			BaseURL:  strings.TrimSpace(os.Getenv(envPrefix(p) + "_BASE_URL")),
			Timeout:  primary.Timeout,
		})
	}
	return out
}

func envPrefix(p Provider) string {
	switch p {
	case ProviderGemini:
		return "GEMINI"
	case ProviderDeepSeek:
		return "DEEPSEEK"
	case ProviderAnthropic:
		return "ANTHROPIC"
	default:
		return "OPENAI"
	}
}

func defaultModel(p Provider) string {
	switch p {
	case ProviderGemini:
		return defaultGeminiModel
	case ProviderDeepSeek:
		return defaultDeepSeekModel
	case ProviderAnthropic:
		return defaultAnthropicModel
	default:
		return defaultOpenAIModel
	}
}

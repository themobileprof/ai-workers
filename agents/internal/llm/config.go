package llm

import "os"

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
	primary, err := New(text)
	if err != nil {
		return Stack{}, err
	}
	s := Stack{Completer: primary, Provider: text.Provider, Model: text.Model}
	if text.Provider == ProviderGemini {
		s.VisionModel = text.Model
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

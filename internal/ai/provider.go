package ai

import (
	"fmt"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	System   string    `json:"system,omitempty"`
}

type ChatResponse struct {
	Content      string        `json:"content"`
	Model        string        `json:"model"`
	Provider     string        `json:"provider"`
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
	Duration     time.Duration `json:"duration_ms"`
}

type Provider interface {
	Chat(req ChatRequest) (*ChatResponse, error)
	Name() string
}

// ProviderConfig holds all provider-specific configuration
type ProviderConfig struct {
	AnthropicKey  string
	GoogleKey     string
	OpenAIKey     string
	DeepSeekKey   string
	OllamaURL     string
	OllamaModel   string
	LocalEndpoint string
	LocalModel    string
	LocalAPIKey   string
}

func NewProvider(name string, cfg ProviderConfig) (Provider, error) {
	switch strings.ToLower(name) {
	case "anthropic", "claude":
		if cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("anthropic API key is required (set ANTHROPIC_API_KEY or in config providers.anthropic_key)")
		}
		return NewAnthropic(cfg.AnthropicKey)
	case "google", "gemini":
		if cfg.GoogleKey == "" {
			return nil, fmt.Errorf("google API key is required (set GOOGLE_API_KEY or in config providers.google_key)")
		}
		return NewGemini(cfg.GoogleKey)
	case "openai", "gpt":
		if cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("openai API key is required (set OPENAI_API_KEY or in config providers.openai_key)")
		}
		return NewOpenAI(cfg.OpenAIKey)
	case "deepseek":
		if cfg.DeepSeekKey == "" {
			return nil, fmt.Errorf("deepseek API key is required (set DEEPSEEK_API_KEY or in config providers.deepseek_key)")
		}
		return NewDeepSeek(cfg.DeepSeekKey)
	case "ollama":
		url := cfg.OllamaURL
		if url == "" {
			url = "http://localhost:11434"
		}
		model := cfg.OllamaModel
		if model == "" {
			model = "qwen2.5"
		}
		return NewOllama(url, model)
	case "local":
		if cfg.LocalEndpoint == "" {
			return nil, fmt.Errorf("local endpoint URL is required (set WEBCLI_LOCAL_ENDPOINT or in config)")
		}
		if cfg.LocalModel == "" {
			return nil, fmt.Errorf("local model name is required (set WEBCLI_LOCAL_MODEL or in config)")
		}
		return NewLocal(cfg.LocalEndpoint, cfg.LocalModel, cfg.LocalAPIKey)
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// DetectProvider auto-detects which provider to use based on available config
func DetectProvider(cfg ProviderConfig) string {
	// Check Ollama first (local, always available if running)
	if cfg.OllamaURL != "" {
		if CheckOllama(cfg.OllamaURL) {
			return "ollama"
		}
	}

	// Check local endpoint
	if cfg.LocalEndpoint != "" && cfg.LocalModel != "" {
		return "local"
	}

	// Check cloud providers
	if cfg.AnthropicKey != "" {
		return "anthropic"
	}
	if cfg.GoogleKey != "" {
		return "google"
	}
	if cfg.OpenAIKey != "" {
		return "openai"
	}
	if cfg.DeepSeekKey != "" {
		return "deepseek"
	}

	return ""
}

func SupportedProviders() []string {
	return []string{
		"ollama (local, any model via Ollama)",
		"local (any OpenAI-compatible endpoint)",
		"anthropic/claude",
		"google/gemini",
		"openai/gpt",
		"deepseek",
	}
}

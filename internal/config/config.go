package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// JSON-friendly duration that marshals as "10s" instead of nanoseconds
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try integer (nanoseconds) for backwards compatibility
		var ns int64
		if err2 := json.Unmarshal(data, &ns); err2 != nil {
			return err
		}
		*d = Duration(time.Duration(ns))
		return nil
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d Duration) ToDuration() time.Duration { return time.Duration(d) }

// ConfigFormat represents the serialization format
type ConfigFormat string

const (
	FormatYAML ConfigFormat = "yaml"
	FormatJSON ConfigFormat = "json"
)

type Config struct {
	Port          int             `yaml:"port" json:"port"`
	Host          string          `yaml:"host" json:"host"`
	UserAgent     string          `yaml:"user_agent" json:"user_agent"`
	SearchTimeout Duration        `yaml:"search_timeout" json:"search_timeout"`
	FetchTimeout  Duration        `yaml:"fetch_timeout" json:"fetch_timeout"`
	MaxResults    int             `yaml:"max_results" json:"max_results"`
	LogLevel      string          `yaml:"log_level" json:"log_level"`
	LogFormat     string          `yaml:"log_format" json:"log_format"`
	RateLimit     RateLimit       `yaml:"rate_limit" json:"rate_limit"`
	Store         StoreConfig     `yaml:"store" json:"store"`
	Search        SearchConfig    `yaml:"search" json:"search"`
	Fetch         FetchConfig     `yaml:"fetch" json:"fetch"`
	Providers     ProvidersConfig `yaml:"providers" json:"providers"`
	configFile    string
}

type ProvidersConfig struct {
	AnthropicKey  string `yaml:"anthropic_key" json:"anthropic_key"`
	GoogleKey     string `yaml:"google_key" json:"google_key"`
	OpenAIKey     string `yaml:"openai_key" json:"openai_key"`
	DeepSeekKey   string `yaml:"deepseek_key" json:"deepseek_key"`
	OllamaURL     string `yaml:"ollama_url" json:"ollama_url"`
	OllamaModel   string `yaml:"ollama_model" json:"ollama_model"`
	LocalEndpoint string `yaml:"local_endpoint" json:"local_endpoint"`
	LocalModel    string `yaml:"local_model" json:"local_model"`
	LocalAPIKey   string `yaml:"local_api_key" json:"local_api_key"`
	SearchAPIKey  string `yaml:"search_api_key" json:"search_api_key"`
	GoogleCX      string `yaml:"google_cx" json:"google_cx"`
}

type RateLimit struct {
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	Burst             int `yaml:"burst" json:"burst"`
}

type StoreConfig struct {
	MemoryFile string `yaml:"memory_file" json:"memory_file"`
	BackupDir  string `yaml:"backup_dir" json:"backup_dir"`
}

type SearchConfig struct {
	Provider    string `yaml:"provider" json:"provider"`
	MaxRetries  int    `yaml:"max_retries" json:"max_retries"`
}

type FetchConfig struct {
	MaxBodySize int `yaml:"max_body_size" json:"max_body_size"`
	MaxTextLen  int `yaml:"max_text_len" json:"max_text_len"`
}

func Default() *Config {
	return &Config{
		Port:          8931,
		Host:          "127.0.0.1",
		UserAgent:     "Mozilla/5.0 (compatible; WebCLI/1.0)",
		SearchTimeout: Duration(10 * time.Second),
		FetchTimeout:  Duration(30 * time.Second),
		MaxResults:    10,
		LogLevel:      "info",
		LogFormat:     "text",
		RateLimit: RateLimit{
			RequestsPerMinute: 30,
			Burst:             5,
		},
		Store: StoreConfig{
			MemoryFile: "",
			BackupDir:  "",
		},
		Search: SearchConfig{
			Provider:    "duckduckgo",
			MaxRetries:  2,
		},
		Fetch: FetchConfig{
			MaxBodySize: 5 * 1024 * 1024,
			MaxTextLen:  10000,
		},
		Providers: ProvidersConfig{
			OllamaURL:   "http://localhost:11434",
			OllamaModel: "qwen2.5",
		},
	}
}

func Load(configPath string) *Config {
	cfg := Default()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			candidates := []string{
				filepath.Join(home, ".webcli", "config.yaml"),
				filepath.Join(home, ".webcli", "config.yml"),
				filepath.Join(home, ".webcli", "config.json"),
				filepath.Join(home, ".config", "webcli", "config.yaml"),
				filepath.Join(home, ".config", "webcli", "config.json"),
			}
			for _, p := range candidates {
				if _, err := os.Stat(p); err == nil {
					configPath = p
					break
				}
			}
		}
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := unmarshalConfig(data, cfg, configPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", configPath, err)
			}
			cfg.configFile = configPath
		}
	}

	cfg.applyEnvOverrides()

	return cfg
}

func (c *Config) Validate() []string {
	var errs []string

	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, "port must be between 1 and 65535")
	}
	if c.SearchTimeout.ToDuration() < 1*time.Second {
		errs = append(errs, "search_timeout must be at least 1s")
	}
	if c.FetchTimeout.ToDuration() < 5*time.Second {
		errs = append(errs, "fetch_timeout must be at least 5s")
	}
	if c.MaxResults < 1 || c.MaxResults > 50 {
		errs = append(errs, "max_results must be between 1 and 50")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error", "":
	default:
		errs = append(errs, "log_level must be one of: debug, info, warn, error")
	}
	switch c.LogFormat {
	case "text", "json", "":
	default:
		errs = append(errs, "log_format must be one of: text, json")
	}
	if c.RateLimit.RequestsPerMinute < 0 {
		errs = append(errs, "rate_limit.requests_per_minute must be >= 0")
	}
	if c.Search.MaxRetries < 0 {
		errs = append(errs, "search.max_retries must be >= 0")
	}
	if c.Fetch.MaxBodySize < 1024 {
		errs = append(errs, "fetch.max_body_size must be at least 1024")
	}
	if c.Fetch.MaxTextLen < 100 {
		errs = append(errs, "fetch.max_text_len must be at least 100")
	}

	return errs
}

func (c *Config) ConfigFile() string {
	return c.configFile
}

func (c *Config) applyEnvOverrides() {
	c.loadDotEnv()

	if v := os.Getenv("WEBCLI_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Port = n
		}
	}
	if v := os.Getenv("WEBCLI_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("WEBCLI_USER_AGENT"); v != "" {
		c.UserAgent = v
	}
	if v := os.Getenv("WEBCLI_SEARCH_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.SearchTimeout = Duration(d)
		}
	}
	if v := os.Getenv("WEBCLI_FETCH_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.FetchTimeout = Duration(d)
		}
	}
	if v := os.Getenv("WEBCLI_MAX_RESULTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxResults = n
		}
	}
	if v := os.Getenv("WEBCLI_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("WEBCLI_LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := os.Getenv("WEBCLI_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RateLimit.RequestsPerMinute = n
		}
	}
	if v := os.Getenv("WEBCLI_MEMORY_FILE"); v != "" {
		c.Store.MemoryFile = v
	}
		if v := os.Getenv("WEBCLI_BACKUP_DIR"); v != "" {
		c.Store.BackupDir = v
	}

	// Provider API keys (from env only — never stored in config file)
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		c.Providers.AnthropicKey = v
	}
	if v := os.Getenv("WEBCLI_ANTHROPIC_KEY"); v != "" {
		c.Providers.AnthropicKey = v
	}
	if v := os.Getenv("GOOGLE_API_KEY"); v != "" {
		c.Providers.GoogleKey = v
	}
	if v := os.Getenv("WEBCLI_GOOGLE_KEY"); v != "" {
		c.Providers.GoogleKey = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		c.Providers.OpenAIKey = v
	}
	if v := os.Getenv("WEBCLI_OPENAI_KEY"); v != "" {
		c.Providers.OpenAIKey = v
	}
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		c.Providers.DeepSeekKey = v
	}
	if v := os.Getenv("WEBCLI_DEEPSEEK_KEY"); v != "" {
		c.Providers.DeepSeekKey = v
	}
	if v := os.Getenv("WEBCLI_SEARCH_API_KEY"); v != "" {
		c.Providers.SearchAPIKey = v
	}
	if v := os.Getenv("WEBCLI_GOOGLE_CX"); v != "" {
		c.Providers.GoogleCX = v
	}
	if v := os.Getenv("OLLAMA_URL"); v != "" {
		c.Providers.OllamaURL = v
	}
	if v := os.Getenv("WEBCLI_OLLAMA_URL"); v != "" {
		c.Providers.OllamaURL = v
	}
	if v := os.Getenv("WEBCLI_OLLAMA_MODEL"); v != "" {
		c.Providers.OllamaModel = v
	}
	if v := os.Getenv("WEBCLI_LOCAL_ENDPOINT"); v != "" {
		c.Providers.LocalEndpoint = v
	}
	if v := os.Getenv("WEBCLI_LOCAL_MODEL"); v != "" {
		c.Providers.LocalModel = v
	}
	if v := os.Getenv("WEBCLI_LOCAL_API_KEY"); v != "" {
		c.Providers.LocalAPIKey = v
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// loadDotEnv loads ~/.webcli/.env if it exists (for cloud API keys)
func (c *Config) loadDotEnv() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	envPath := filepath.Join(home, ".webcli", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		parts := strings.SplitN(line[7:], "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".webcli", "config.yaml")
}

func ConfigPathJSON() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".webcli", "config.json")
}

// DetectFormat returns the format based on file extension
func DetectFormat(path string) ConfigFormat {
	switch {
	case strings.HasSuffix(path, ".json"):
		return FormatJSON
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return FormatYAML
	default:
		return FormatYAML
	}
}

func unmarshalConfig(data []byte, cfg *Config, path string) error {
	switch DetectFormat(path) {
	case FormatJSON:
		return json.Unmarshal(data, cfg)
	default:
		return yaml.Unmarshal(data, cfg)
	}
}

func WriteDefaultConfig(path string) error {
	cfg := Default()
	// Only write non-secret provider defaults (Ollama URL)
	// API keys for cloud providers must be set via env vars for security
	cfgCopy := *cfg
	cfgCopy.Providers = ProvidersConfig{
		OllamaURL:   cfg.Providers.OllamaURL,
		OllamaModel: cfg.Providers.OllamaModel,
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	switch DetectFormat(path) {
	case FormatJSON:
		data, err := json.MarshalIndent(&cfgCopy, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		return os.WriteFile(path, data, 0644)
	default:
		data, err := yaml.Marshal(&cfgCopy)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}
		return os.WriteFile(path, data, 0644)
	}
}

// maskKeys clones the config and masks all secret values
func maskKeys(cfg *Config) *Config {
	display := *cfg
	if display.Providers.AnthropicKey != "" {
		display.Providers.AnthropicKey = maskKey(display.Providers.AnthropicKey)
	}
	if display.Providers.GoogleKey != "" {
		display.Providers.GoogleKey = maskKey(display.Providers.GoogleKey)
	}
	if display.Providers.OpenAIKey != "" {
		display.Providers.OpenAIKey = maskKey(display.Providers.OpenAIKey)
	}
	if display.Providers.DeepSeekKey != "" {
		display.Providers.DeepSeekKey = maskKey(display.Providers.DeepSeekKey)
	}
	if display.Providers.SearchAPIKey != "" {
		display.Providers.SearchAPIKey = maskKey(display.Providers.SearchAPIKey)
	}
	if display.Providers.LocalAPIKey != "" {
		display.Providers.LocalAPIKey = maskKey(display.Providers.LocalAPIKey)
	}
	return &display
}

func FormatConfig(cfg *Config) (string, error) {
	display := maskKeys(cfg)
	data, err := yaml.Marshal(display)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FormatConfigJSON(cfg *Config) (string, error) {
	display := maskKeys(cfg)
	data, err := json.MarshalIndent(display, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func parseEnvList(val string) []string {
	if val == "" {
		return nil
	}
	var result []string
	for _, s := range strings.Split(val, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

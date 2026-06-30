package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/ai"
	"github.com/yenya/webcli/internal/config"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run setup wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetup()
	},
}

func needsSetup() bool {
	home, _ := os.UserHomeDir()
	if home == "" {
		return false
	}

	// If any config file exists, assume configured
	for _, p := range []string{
		filepath.Join(home, ".webcli", "config.yaml"),
		filepath.Join(home, ".webcli", "config.yml"),
		filepath.Join(home, ".webcli", "config.json"),
	} {
		if _, err := os.Stat(p); err == nil {
			return false
		}
	}

	// If any API key is set in env, assume configured
	keys := []string{
		"ANTHROPIC_API_KEY", "GOOGLE_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY",
		"WEBCLI_ANTHROPIC_KEY", "WEBCLI_GOOGLE_KEY", "WEBCLI_OPENAI_KEY", "WEBCLI_DEEPSEEK_KEY",
		"WEBCLI_LOCAL_ENDPOINT", "WEBCLI_SEARCH_API_KEY",
	}
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return false
		}
	}

	// If Ollama is running, assume configured
	if ai.CheckOllama("http://localhost:11434") {
		return false
	}

	return true
}

func prompt(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func promptDefault(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func promptYN(label string, def bool) bool {
	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Printf("%s %s: ", label, suffix)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))
	switch text {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

func runSetup() error {
	fmt.Println()
	fmt.Println("╭────────────────────────────────────────────╮")
	fmt.Println("│       WebCLI Setup Wizard                 │")
	fmt.Println("│                                           │")
	fmt.Println("│  Let's get you connected to an AI model.  │")
	fmt.Println("│  Pick one (or more) of the options below. │")
	fmt.Println("╰────────────────────────────────────────────╯")
	fmt.Println()

	providers := config.ProvidersConfig{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "qwen2.5",
	}

	// 1. Check Ollama
	ollamaRunning := ai.CheckOllama(providers.OllamaURL)
	if ollamaRunning {
		fmt.Println("✓ Ollama detected at", providers.OllamaURL)
		m := promptDefault("Default Ollama model", providers.OllamaModel)
		if m != "" {
			providers.OllamaModel = m
		}
	} else {
		fmt.Println("✗ Ollama not detected at", providers.OllamaURL)
		url := promptDefault("Ollama URL (or leave empty to skip)", "")
		if url != "" {
			providers.OllamaURL = url
			if ai.CheckOllama(url) {
				fmt.Println("✓ Ollama detected!")
				m := promptDefault("Default Ollama model", "qwen2.5")
				if m != "" {
					providers.OllamaModel = m
				}
			} else {
				fmt.Println("  (Ollama not reachable at that URL either — you can still configure it)")
				providers.OllamaModel = promptDefault("Ollama model name", "qwen2.5")
			}
		} else {
			providers.OllamaURL = ""
			providers.OllamaModel = ""
		}
	}

	// 2. Local endpoint
	fmt.Println()
	if promptYN("Configure an OpenAI-compatible local endpoint?", false) {
		providers.LocalEndpoint = promptDefault("Endpoint URL (e.g. http://localhost:8000/v1/chat/completions)", "")
		providers.LocalModel = promptDefault("Model name", "")
		if promptYN("Does this endpoint require an API key?", false) {
			providers.LocalAPIKey = prompt("API key")
		}
	}

	// 3. Cloud API keys
	fmt.Println()
	fmt.Println("Cloud providers (optional — set API keys via env vars):")
	fmt.Println()

	keys := map[string]*string{
		"Anthropic (Claude)":   &cfg.Providers.AnthropicKey,
		"Google (Gemini)":      &cfg.Providers.GoogleKey,
		"OpenAI (GPT)":         &cfg.Providers.OpenAIKey,
		"DeepSeek":             &cfg.Providers.DeepSeekKey,
	}

	for name, ptr := range keys {
		if promptYN(fmt.Sprintf("Set %s API key?", name), false) {
			val := prompt("  API key (will be stored in env, not config)")
			if val != "" {
				*ptr = val
			}
		}
	}

	// 4. Search API key
	fmt.Println()
	if promptYN("Configure Tavily search API key? (DuckDuckGo is free, no key needed)", false) {
		providers.SearchAPIKey = prompt("  Tavily API key")
	}

	// 5. Write config
	fmt.Println()
	writeConfig := ollamaRunning || providers.LocalEndpoint != "" || providers.LocalAPIKey != ""

	if !writeConfig {
		// Even with nothing configured, write a minimal config so first-run check doesn't repeat
		home, _ := os.UserHomeDir()
		cfgPath := filepath.Join(home, ".webcli", "config.json")
		os.MkdirAll(filepath.Dir(cfgPath), 0755)
		os.WriteFile(cfgPath, []byte("{}\n"), 0644)
		fmt.Println("✓ Minimal config written to", cfgPath)
		fmt.Println("  (run 'webcli setup' anytime to add providers)")
	} else if promptYN("Write config file?", true) {
		home, _ := os.UserHomeDir()
		cfgPath := filepath.Join(home, ".webcli", "config.json")

		cfgData := map[string]interface{}{
			"providers": map[string]interface{}{
				"ollama_url":      providers.OllamaURL,
				"ollama_model":    providers.OllamaModel,
				"local_endpoint":  providers.LocalEndpoint,
				"local_model":     providers.LocalModel,
				"local_api_key":   providers.LocalAPIKey,
				"search_api_key":  providers.SearchAPIKey,
			},
		}

		if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		data, _ := json.MarshalIndent(cfgData, "", "  ")
		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Println("✓ Config written to", cfgPath)
	}

	// 6. Provide .env hint for cloud keys
	hasCloudKeys := false
	for _, ptr := range keys {
		if *ptr != "" {
			hasCloudKeys = true
			break
		}
	}
	if hasCloudKeys {
		home, _ := os.UserHomeDir()
		envPath := filepath.Join(home, ".webcli", ".env")
		if promptYN("Save cloud API keys to ~/.webcli/.env?", true) {
			var lines []string
			if cfg.Providers.AnthropicKey != "" {
				lines = append(lines, "export ANTHROPIC_API_KEY="+cfg.Providers.AnthropicKey)
			}
			if cfg.Providers.GoogleKey != "" {
				lines = append(lines, "export GOOGLE_API_KEY="+cfg.Providers.GoogleKey)
			}
			if cfg.Providers.OpenAIKey != "" {
				lines = append(lines, "export OPENAI_API_KEY="+cfg.Providers.OpenAIKey)
			}
			if cfg.Providers.DeepSeekKey != "" {
				lines = append(lines, "export DEEPSEEK_API_KEY="+cfg.Providers.DeepSeekKey)
			}
			data := []byte(strings.Join(lines, "\n") + "\n")
			os.WriteFile(envPath, data, 0600)
			fmt.Println("✓ Keys saved to", envPath)
			fmt.Println("  Source it with: source " + envPath)
			fmt.Println("  Or add the exports to your ~/.zshrc / ~/.bashrc")
		}
		fmt.Println()
		fmt.Println("⚠  Cloud API keys are sensitive. Never commit them to git.")
		fmt.Println("   The .env file is gitignored by default.")
	}

	// 7. Summary
	fmt.Println()
	fmt.Println("╭────────────────────────────────────────────╮")
	fmt.Println("│              Setup Complete!              │")
	fmt.Println("╰────────────────────────────────────────────╯")
	fmt.Println()
	fmt.Println("Try these commands:")
	if ollamaRunning || providers.LocalEndpoint != "" {
		fmt.Println("  webcli chat \"Hello, world!\"")
	}
	if hasCloudKeys {
		fmt.Println("  webcli chat --provider claude \"Hello\"")
		fmt.Println("  webcli chat --provider gemini \"Hello\"")
	}
	fmt.Println("  webcli search \"latest AI news\"")
	fmt.Println("  webcli serve --transport sse")
	fmt.Println()

	return nil
}

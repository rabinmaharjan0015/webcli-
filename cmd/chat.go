package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/ai"
	"github.com/yenya/webcli/internal/log"
	"github.com/yenya/webcli/internal/store"
)

var chatProvider string
var chatModel string
var chatSystem string
var chatWithContext bool

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&chatProvider, "provider", "", "", "AI provider: ollama, local, anthropic, google, openai, deepseek")
	chatCmd.Flags().StringVarP(&chatModel, "model", "M", "", "Model name (defaults per provider)")
	chatCmd.Flags().StringVarP(&chatSystem, "system", "s", "", "System prompt")
	chatCmd.Flags().BoolVarP(&chatWithContext, "context", "c", false, "Inject shared memory context into system prompt")
}

var chatCmd = &cobra.Command{
	Use:   "chat [--provider ollama|local|anthropic|google|openai|deepseek] <prompt>",
	Short: "Chat with an AI model using your own API key or local model",
	Long: `Send a prompt to an AI model (Claude, Gemini, GPT, DeepSeek, or local models via Ollama/OpenAI-compatible endpoint).

Configure keys via environment variables (recommended) or config file:

  ANTHROPIC_API_KEY or WEBCLI_ANTHROPIC_KEY   for Claude
  GOOGLE_API_KEY    or WEBCLI_GOOGLE_KEY       for Gemini
  OPENAI_API_KEY    or WEBCLI_OPENAI_KEY       for GPT
  DEEPSEEK_API_KEY  or WEBCLI_DEEPSEEK_KEY     for DeepSeek

Local models via Ollama (no API key needed):
  OLLAMA_URL        (default: http://localhost:11434)
  WEBCLI_OLLAMA_MODEL (default: qwen2.5)

Local models via OpenAI-compatible endpoint:
  WEBCLI_LOCAL_ENDPOINT  (e.g. http://localhost:8000/v1/chat/completions)
  WEBCLI_LOCAL_MODEL     (e.g. qwen2.5-7b-instruct)
  WEBCLI_LOCAL_API_KEY   (optional)`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")

		providerCfg := ai.ProviderConfig{
			AnthropicKey:  cfg.Providers.AnthropicKey,
			GoogleKey:     cfg.Providers.GoogleKey,
			OpenAIKey:     cfg.Providers.OpenAIKey,
			DeepSeekKey:   cfg.Providers.DeepSeekKey,
			OllamaURL:     cfg.Providers.OllamaURL,
			OllamaModel:   cfg.Providers.OllamaModel,
			LocalEndpoint: cfg.Providers.LocalEndpoint,
			LocalModel:    cfg.Providers.LocalModel,
			LocalAPIKey:   cfg.Providers.LocalAPIKey,
		}

		providerName := chatProvider
		if providerName == "" {
			providerName = ai.DetectProvider(providerCfg)
			if providerName == "" {
				return fmt.Errorf("no provider available. "+
					"Set an API key via env var for cloud providers, "+
					"or run a local model via Ollama (http://localhost:11434).\n"+
					"Supported providers:\n"+
					"  ollama (local)  — set OLLAMA_URL or WEBCLI_OLLAMA_MODEL\n"+
					"  local           — set WEBCLI_LOCAL_ENDPOINT and WEBCLI_LOCAL_MODEL\n"+
					"  anthropic       — set ANTHROPIC_API_KEY\n"+
					"  google          — set GOOGLE_API_KEY\n"+
					"  openai          — set OPENAI_API_KEY\n"+
					"  deepseek        — set DEEPSEEK_API_KEY")
			}
		}

		provider, err := ai.NewProvider(providerName, providerCfg)
		if err != nil {
			return fmt.Errorf("init provider: %w", err)
		}

		systemPrompt := chatSystem
		if chatWithContext {
			s, err := store.New(cfg.Store.MemoryFile)
			if err == nil {
				context := s.GetContext("")
				if context != "" {
					if systemPrompt != "" {
						systemPrompt += "\n\n"
					}
					systemPrompt += context
				}
			}
		}

		log.Info("Sending to %s...", providerName)
		resp, err := provider.Chat(ai.ChatRequest{
			Model:  chatModel,
			System: systemPrompt,
			Messages: []ai.Message{
				{Role: "user", Content: prompt},
			},
		})
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

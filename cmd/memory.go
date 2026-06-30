package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/store"
)

var memoryFile string

func storePath() string {
	if memoryFile != "" {
		return memoryFile
	}
	return cfg.Store.MemoryFile
}

func init() {
	rootCmd.AddCommand(memoryCmd)
	memoryCmd.AddCommand(rememberCmd)
	memoryCmd.AddCommand(recallCmd)
	memoryCmd.AddCommand(forgetCmd)
	memoryCmd.AddCommand(contextCmd)

	for _, c := range []*cobra.Command{rememberCmd, recallCmd, forgetCmd, contextCmd} {
		c.Flags().StringVarP(&memoryFile, "file", "f", "", "memory store file path")
	}
	rememberCmd.Flags().StringP("category", "c", "fact", "category: fact, discovery, preference, instruction")
	rememberCmd.Flags().StringP("tags", "t", "", "comma-separated tags")
	contextCmd.Flags().StringP("topic", "q", "", "topic to get context about (leave empty for all)")
}

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Shared knowledge across AI models",
	Long: `Persistent knowledge store shared across AI models (Claude, DeepSeek, Qwen, etc.).
One model's discoveries are immediately available to all others.

Categories: fact (default), discovery, preference, instruction

Commands:
  remember <key> <value>   save knowledge
  recall   <key>           retrieve knowledge
  forget   <key>           delete knowledge
  context                 get relevant shared knowledge (for prompt injection)`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var rememberCmd = &cobra.Command{
	Use:   "remember <key> <value>",
	Short: "Save knowledge to shared memory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(storePath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		catStr, _ := cmd.Flags().GetString("category")
		cat := store.CategoryFact
		switch store.Category(catStr) {
		case store.CategoryDiscovery, store.CategoryPreference, store.CategoryInstruction:
			cat = store.Category(catStr)
		}

		tagsStr, _ := cmd.Flags().GetString("tags")
		var tags []string
		if tagsStr != "" {
			for _, t := range splitAndTrim(tagsStr, ",") {
				tags = append(tags, t)
			}
		}

		item, err := s.Save(args[0], args[1], store.WithCategory(cat), store.WithTags(tags))
		if err != nil {
			return fmt.Errorf("save: %w", err)
		}

		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

var recallCmd = &cobra.Command{
	Use:   "recall <key>",
	Short: "Retrieve knowledge from shared memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(storePath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		item, err := s.Get(args[0])
		if err != nil {
			return fmt.Errorf("recall: %w", err)
		}

		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

var forgetCmd = &cobra.Command{
	Use:   "forget <key>",
	Short: "Delete knowledge from shared memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(storePath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := s.Delete(args[0]); err != nil {
			return fmt.Errorf("forget: %w", err)
		}

		fmt.Printf("memory '%s' deleted\n", args[0])
		return nil
	},
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Get shared knowledge context for prompt injection",
	Long: `Outputs relevant shared knowledge as formatted text.
Pipe this into another model's prompt to share what other models have learned.

Examples:
  webcli memory context                     # all knowledge
  webcli memory context -q "python"         # knowledge about python
  webcli memory context -q "golang" | xsel  # pipe to clipboard`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(storePath())
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		topic, _ := cmd.Flags().GetString("topic")
		context := s.GetContext(topic)

		if context == "" {
			if topic != "" {
				fmt.Printf("No shared knowledge found about \"%s\".\n", topic)
				return nil
			}
			fmt.Println("No shared knowledge stored yet.")
			return nil
		}

		fmt.Println(context)
		return nil
	},
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

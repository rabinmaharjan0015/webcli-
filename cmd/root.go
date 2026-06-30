package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/config"
	"github.com/yenya/webcli/internal/log"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "webcli",
	Short: "WebCLI - Web access & shared memory for AI agents",
	Long: `WebCLI provides web search, page fetching, and persistent shared memory
for AI agents (Claude, DeepSeek, Qwen, etc.). Multiple models can connect
to the same MCP server and learn from each other.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cfg == nil {
			cfg = config.Load("")
		}
		if cmd.Use != "completion" && cmd.Use != "help" && cmd.Use != "setup" {
			log.Init(cfg.LogLevel, cfg.LogFormat)
			if needsSetup() {
				fmt.Println()
				fmt.Println("⚡ First run detected! Let's get you set up.")
				fmt.Println("   (Run 'webcli setup' later to reconfigure)")
				fmt.Println()
				runSetup()
				// Reload config after setup
				cfg = config.Load(cfg.ConfigFile())
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cfg = config.Load("")

	rootCmd.PersistentFlags().IntVarP(&cfg.Port, "port", "p", cfg.Port, "MCP server port")
	rootCmd.PersistentFlags().IntVarP(&cfg.MaxResults, "max-results", "n", cfg.MaxResults, "max search results")
	rootCmd.PersistentFlags().StringVarP(&cfg.LogLevel, "log-level", "l", cfg.LogLevel, "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "log format: text or json")
	rootCmd.PersistentFlags().StringVarP(&cfg.Store.MemoryFile, "memory-file", "m", cfg.Store.MemoryFile, "memory store file path")
}

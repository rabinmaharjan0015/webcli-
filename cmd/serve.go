package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/log"
	"github.com/yenya/webcli/internal/server"
)

var transport string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run MCP server for AI agents",
	Long: `Start the MCP (Model Context Protocol) server for AI agents.

SSE mode (default): starts an HTTP server with MCP endpoints at /sse and /message.
Stdio mode: used by AI IDEs/agents that spawn the process directly.

The server loads config from ~/.webcli/config.yaml, env vars, and CLI flags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if errs := cfg.Validate(); len(errs) > 0 {
			for _, e := range errs {
				log.Error("config: %s", e)
			}
			return fmt.Errorf("invalid configuration (%d errors)", len(errs))
		}

		if f := cfg.ConfigFile(); f != "" {
			log.Info("Loaded config from %s", f)
		}
		log.Info("Shared memory: %s", cfg.Store.MemoryFile)

		return server.RunMCPServer(cfg, transport)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVarP(&transport, "transport", "t", "sse", "transport mode: sse or stdio")
}

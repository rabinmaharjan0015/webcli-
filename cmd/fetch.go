package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/log"
	"github.com/yenya/webcli/internal/web"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <url>",
	Short: "Fetch and extract text content from a URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawURL := args[0]
		if rawURL == "" {
			return fmt.Errorf("url cannot be empty")
		}
		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			return fmt.Errorf("url must start with http:// or https://")
		}

		log.Info("Fetching: %s", rawURL)
		rateLimiter := web.NewRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)
		fetcher := web.NewFetcher(cfg.FetchTimeout.ToDuration(), cfg.UserAgent, rateLimiter)

		result, err := fetcher.Fetch(rawURL)
		if err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encode result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}

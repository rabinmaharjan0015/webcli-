package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/log"
	"github.com/yenya/webcli/internal/web"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the web",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		log.Info("Searching for: %s", query)
		rateLimiter := web.NewRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)
		searcher, err := web.NewSearchProvider(
			cfg.Search.Provider,
			cfg.Providers.SearchAPIKey,
			cfg.Providers.GoogleCX,
			rateLimiter,
			cfg.UserAgent,
		)
		if err != nil {
			return fmt.Errorf("search provider: %w", err)
		}

		results, err := searcher.Search(query, cfg.MaxResults)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("encode results: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

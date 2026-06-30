package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/config"
)

var configShowFormat string
var configInitJSON bool

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)

	configShowCmd.Flags().StringVarP(&configShowFormat, "format", "f", "yaml", "output format: yaml or json")
	configInitCmd.Flags().BoolVarP(&configInitJSON, "json", "j", false, "write JSON config file instead of YAML")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		var out string
		var err error
		switch strings.ToLower(configShowFormat) {
		case "json":
			out, err = config.FormatConfigJSON(cfg)
		default:
			out, err = config.FormatConfig(cfg)
		}
		if err != nil {
			return fmt.Errorf("format: %w", err)
		}
		fmt.Print(out)
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		errs := cfg.Validate()
		if len(errs) == 0 {
			if f := cfg.ConfigFile(); f != "" {
				fmt.Printf("Config %s: valid\n", f)
			} else {
				fmt.Println("Config (defaults): valid")
			}
			return nil
		}
		if f := cfg.ConfigFile(); f != "" {
			fmt.Printf("Config %s: %d error(s)\n", f, len(errs))
		} else {
			fmt.Printf("Config: %d error(s)\n", len(errs))
		}
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return fmt.Errorf("validation failed (%d errors)", len(errs))
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if configInitJSON {
			path = config.ConfigPathJSON()
		} else {
			path = config.ConfigPath()
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s", path)
		}
		if err := config.WriteDefaultConfig(path); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("Default config written to %s\n", path)
		return nil
	},
}

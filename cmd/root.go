package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/typhonius/pidge/internal/config"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	configPath string
	cfg        *config.Config
	client     *smsgateway.Client
)

var rootCmd = &cobra.Command{
	Use:   "pidge",
	Short: "CLI for Android SMS Gateway",
	Long:  "A command-line tool for interacting with Android SMS Gateway's local API.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for setup command.
		if cmd.Name() == "setup" {
			return nil
		}
		return loadConfig()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default ~/.config/pidge/config.toml)")
}

func loadConfig() error {
	path := configPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return err
		}
		path = p
	}

	c, err := config.Load(path)
	if err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "No config file found. Run 'pidge setup' to create one.")
			os.Exit(1)
		}
		return fmt.Errorf("loading config: %w", err)
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	cfg = c
	sdkConfig := smsgateway.Config{}.
		WithBaseURL(cfg.Gateway.URL).
		WithBasicAuth(cfg.Gateway.Username, cfg.Gateway.Password)
	client = smsgateway.NewClient(sdkConfig)
	return nil
}

// printJSON marshals v to JSON and writes to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

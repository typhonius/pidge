package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(healthCmd)
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check gateway health",
	RunE:  runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.CheckHealth(ctx)
	if err != nil {
		return fmt.Errorf("checking health: %w", err)
	}

	if jsonOutput {
		return printJSON(health)
	}

	fmt.Printf("Status:  %s\n", health.Status)
	fmt.Printf("Version: %s\n", health.Version)
	if len(health.Checks) > 0 {
		fmt.Println("\nChecks:")
		for name, check := range health.Checks {
			fmt.Printf("  %-20s %s (%d %s)\n", name, check.Status, check.ObservedValue, check.ObservedUnit)
		}
	}
	return nil
}

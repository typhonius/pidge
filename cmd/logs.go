package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View device logs",
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch logs from the last 24 hours.
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	entries, err := client.GetLogs(ctx, from, now)
	if err != nil {
		return fmt.Errorf("fetching logs: %w", err)
	}

	if jsonOutput {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Println("No log entries.")
		return nil
	}

	for _, e := range entries {
		fmt.Printf("[%s] %-5s %-15s %s\n",
			e.CreatedAt.Local().Format("2006-01-02 15:04:05"),
			e.Priority,
			e.Module,
			e.Message,
		)
	}
	return nil
}

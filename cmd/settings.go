package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(settingsCmd)
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "View device settings",
	RunE:  runSettings,
}

func runSettings(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := client.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("fetching settings: %w", err)
	}

	if jsonOutput {
		return printJSON(settings)
	}

	if settings.Messages != nil {
		fmt.Println("Messages:")
		m := settings.Messages
		if m.LimitPeriod != nil {
			fmt.Printf("  Limit period:      %s\n", *m.LimitPeriod)
		}
		if m.LimitValue != nil {
			fmt.Printf("  Limit value:       %d\n", *m.LimitValue)
		}
		if m.SendIntervalMin != nil {
			fmt.Printf("  Send interval min: %d\n", *m.SendIntervalMin)
		}
		if m.SendIntervalMax != nil {
			fmt.Printf("  Send interval max: %d\n", *m.SendIntervalMax)
		}
		if m.SimSelectionMode != nil {
			fmt.Printf("  SIM selection:     %s\n", *m.SimSelectionMode)
		}
		if m.LogLifetimeDays != nil {
			fmt.Printf("  Log lifetime:      %d days\n", *m.LogLifetimeDays)
		}
		if m.ProcessingOrder != nil {
			fmt.Printf("  Processing order:  %s\n", *m.ProcessingOrder)
		}
	}

	if settings.Ping != nil && settings.Ping.IntervalSeconds != nil {
		fmt.Printf("\nPing:\n  Interval: %ds\n", *settings.Ping.IntervalSeconds)
	}

	if settings.Logs != nil && settings.Logs.LifetimeDays != nil {
		fmt.Printf("\nLogs:\n  Lifetime: %d days\n", *settings.Logs.LifetimeDays)
	}

	if settings.Webhooks != nil {
		fmt.Println("\nWebhooks:")
		w := settings.Webhooks
		if w.InternetRequired != nil {
			fmt.Printf("  Internet required: %v\n", *w.InternetRequired)
		}
		if w.RetryCount != nil {
			fmt.Printf("  Retry count:       %d\n", *w.RetryCount)
		}
	}

	if settings.Gateway != nil {
		fmt.Println("\nGateway:")
		g := settings.Gateway
		if g.CloudURL != nil {
			fmt.Printf("  Cloud URL: %s\n", *g.CloudURL)
		}
	}

	return nil
}

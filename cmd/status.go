package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <message-id>",
	Short: "Check message delivery status",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	state, err := client.GetState(ctx, id)
	if err != nil {
		return fmt.Errorf("getting message state: %w", err)
	}

	if jsonOutput {
		return printJSON(state)
	}

	fmt.Printf("Message: %s\n", state.ID)
	fmt.Printf("State:   %s\n", state.State)
	fmt.Println("\nRecipients:")
	for _, r := range state.Recipients {
		line := fmt.Sprintf("  %s: %s", r.PhoneNumber, r.State)
		if r.Error != nil {
			line += fmt.Sprintf(" (error: %s)", *r.Error)
		}
		fmt.Println(line)
	}
	return nil
}

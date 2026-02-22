package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:   "send <number> <message...>",
	Short: "Send an SMS",
	Long:  "Send a text message to the specified phone number.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runSend,
}

func runSend(cmd *cobra.Command, args []string) error {
	number := args[0]
	message := strings.Join(args[1:], " ")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	state, err := client.Send(ctx, smsgateway.Message{
		TextMessage:  &smsgateway.TextMessage{Text: message},
		PhoneNumbers: []string{number},
	})
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	if jsonOutput {
		return printJSON(state)
	}

	fmt.Printf("Message sent (ID: %s)\n", state.ID)
	fmt.Printf("State: %s\n", state.State)
	for _, r := range state.Recipients {
		fmt.Printf("  %s: %s\n", r.PhoneNumber, r.State)
	}
	return nil
}

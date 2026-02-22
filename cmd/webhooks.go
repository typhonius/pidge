package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(webhooksListCmd)
	webhooksCmd.AddCommand(webhooksAddCmd)
	webhooksCmd.AddCommand(webhooksDeleteCmd)
}

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhooks",
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered webhooks",
	RunE:  runWebhooksList,
}

var webhooksAddCmd = &cobra.Command{
	Use:   "add <url> <event>",
	Short: "Register a webhook",
	Long: fmt.Sprintf("Register a webhook for the given URL and event type.\n\nValid events: %s",
		"sms:received, sms:sent, sms:delivered, sms:failed, sms:data-received, mms:received, system:ping"),
	Args: cobra.ExactArgs(2),
	RunE: runWebhooksAdd,
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDelete,
}

func runWebhooksList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hooks, err := client.ListWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("listing webhooks: %w", err)
	}

	if jsonOutput {
		return printJSON(hooks)
	}

	if len(hooks) == 0 {
		fmt.Println("No webhooks registered.")
		return nil
	}

	for _, h := range hooks {
		fmt.Printf("%-36s  %-20s  %s\n", h.ID, h.Event, h.URL)
	}
	return nil
}

func runWebhooksAdd(cmd *cobra.Command, args []string) error {
	url := args[0]
	event := smsgateway.WebhookEvent(args[1])

	if !smsgateway.IsValidWebhookEvent(event) {
		return fmt.Errorf("invalid event type %q; valid types: sms:received, sms:sent, sms:delivered, sms:failed, sms:data-received, mms:received, system:ping", args[1])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hook, err := client.RegisterWebhook(ctx, smsgateway.Webhook{
		URL:   url,
		Event: event,
	})
	if err != nil {
		return fmt.Errorf("registering webhook: %w", err)
	}

	if jsonOutput {
		return printJSON(hook)
	}

	fmt.Printf("Webhook registered (ID: %s)\n", hook.ID)
	fmt.Printf("  URL:   %s\n", hook.URL)
	fmt.Printf("  Event: %s\n", hook.Event)
	return nil
}

func runWebhooksDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.DeleteWebhook(ctx, id); err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}

	if jsonOutput {
		return printJSON(map[string]string{"deleted": id})
	}

	fmt.Printf("Webhook %s deleted.\n", id)
	return nil
}

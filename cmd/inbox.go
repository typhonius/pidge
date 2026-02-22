package cmd

import (
	"fmt"
	"os"

	"github.com/typhonius/pidge/internal/store"
	"github.com/spf13/cobra"
)

var unreadOnly bool

func init() {
	inboxCmd.Flags().BoolVar(&unreadOnly, "unread", false, "only show unprocessed messages")
	rootCmd.AddCommand(inboxCmd)
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List received messages",
	Long:  "List messages received via webhooks. Requires 'pidge serve' to be running to capture incoming SMS.",
	RunE:  runInbox,
}

func runInbox(cmd *cobra.Command, args []string) error {
	dbPath := cfg.ExpandDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("no message store found at %s — is 'pidge serve' running?", dbPath)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer st.Close()

	f := store.ListFilter{Limit: 50}
	if unreadOnly {
		unread := false
		f.Processed = &unread
	}
	messages, err := st.ListMessages(f)
	if err != nil {
		return fmt.Errorf("listing messages: %w", err)
	}

	if jsonOutput {
		return printJSON(messages)
	}

	if len(messages) == 0 {
		fmt.Println("No messages.")
		return nil
	}

	for _, m := range messages {
		body := m.Message
		if len(body) > 60 {
			body = body[:57] + "..."
		}
		status := " "
		if m.Processed {
			status = "+"
		}
		fmt.Printf("[%s] %3d  %-14s  %s  %s\n",
			status,
			m.ID,
			m.PhoneNumber,
			m.ReceivedAt.Local().Format("2006-01-02 15:04"),
			body,
		)
	}
	return nil
}

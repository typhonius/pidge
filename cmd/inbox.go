package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/typhonius/pidge/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(inboxCmd)
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List received messages",
	Long:  "List messages received by the gateway device. Reads from local store if available, otherwise falls back to the gateway API.",
	RunE:  runInbox,
}

// inboxMessage represents a received message from the gateway API.
type inboxMessage struct {
	ID           string                      `json:"id"`
	Message      string                      `json:"message"`
	PhoneNumbers []string                    `json:"phoneNumbers"`
	State        smsgateway.ProcessingState  `json:"state"`
	Recipients   []smsgateway.RecipientState `json:"recipients"`
	CreatedAt    time.Time                   `json:"createdAt"`
}

func runInbox(cmd *cobra.Command, args []string) error {
	// Try local store first
	dbPath := cfg.ExpandDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		return runInboxFromStore(dbPath)
	}

	slog.Debug("local store not found, falling back to gateway API", "path", dbPath)
	return runInboxFromGateway()
}

func runInboxFromStore(dbPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer st.Close()

	messages, err := st.ListMessages(store.ListFilter{Limit: 50})
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
		fmt.Printf("[%s] %-14s  %s  %s\n",
			status,
			m.PhoneNumber,
			m.ReceivedAt.Local().Format("2006-01-02 15:04"),
			body,
		)
	}
	return nil
}

func runInboxFromGateway() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := cfg.Gateway.URL + "/api/v1/message"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(cfg.Gateway.Username, cfg.Gateway.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching inbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("inbox request failed (%d): %s", resp.StatusCode, string(body))
	}

	var messages []inboxMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return fmt.Errorf("decoding inbox: %w", err)
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
		from := ""
		if len(m.PhoneNumbers) > 0 {
			from = m.PhoneNumbers[0]
		}
		fmt.Printf("%-14s  %-10s  %s  %s\n",
			from,
			m.State,
			m.CreatedAt.Local().Format("2006-01-02 15:04"),
			body,
		)
	}
	return nil
}

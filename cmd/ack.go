package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	ackServer string
	ackAll    bool
)

func init() {
	ackCmd.Flags().StringVar(&ackServer, "server", "", "pidge server URL (default: derived from webhook_url or http://localhost:3851)")
	ackCmd.Flags().BoolVar(&ackAll, "all", false, "mark all messages as processed")
	rootCmd.AddCommand(ackCmd)
}

var ackCmd = &cobra.Command{
	Use:   "ack [id]",
	Short: "Mark a message as processed",
	Long:  "Mark a received message (or all messages with --all) as processed on the pidge server.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAck,
}

func runAck(cmd *cobra.Command, args []string) error {
	if !ackAll && len(args) == 0 {
		return fmt.Errorf("provide a message id or use --all")
	}

	base, err := resolveServerURL(ackServer)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}

	if ackAll {
		endpoint := fmt.Sprintf("%s/api/messages/processed", strings.TrimRight(base, "/"))
		resp, err := client.Post(endpoint, "", nil)
		if err != nil {
			return fmt.Errorf("reaching pidge server: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		if jsonOutput {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			return nil
		}

		fmt.Println("All messages marked as processed.")
		return nil
	}

	id := args[0]
	endpoint := fmt.Sprintf("%s/api/messages/%s/processed", strings.TrimRight(base, "/"), id)

	resp, err := client.Post(endpoint, "", nil)
	if err != nil {
		return fmt.Errorf("reaching pidge server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("message %s not found", id)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	if jsonOutput {
		return printJSON(map[string]string{"status": "ok", "id": id})
	}

	fmt.Printf("Message %s marked as processed.\n", id)
	return nil
}

// resolveServerURL determines the pidge server base URL from an explicit
// override, the webhook_url config, or the listen address.
func resolveServerURL(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	if cfg.Server.WebhookURL != "" {
		u, err := url.Parse(cfg.Server.WebhookURL)
		if err == nil && u.Host != "" {
			return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
		}
	}

	listen := cfg.Server.Listen
	if listen == "" {
		listen = ":3851"
	}
	host := listen
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	return "http://" + host, nil
}

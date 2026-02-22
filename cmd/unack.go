package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var unackServer string

func init() {
	unackCmd.Flags().StringVar(&unackServer, "server", "", "pidge server URL (default: derived from webhook_url or http://localhost:3851)")
	rootCmd.AddCommand(unackCmd)
}

var unackCmd = &cobra.Command{
	Use:   "unack <id>",
	Short: "Mark a message as unprocessed",
	Long:  "Mark a received message as unprocessed on the pidge server.",
	Args:  cobra.ExactArgs(1),
	RunE:  runUnack,
}

func runUnack(cmd *cobra.Command, args []string) error {
	server, err := resolveServerURL(unackServer)
	if err != nil {
		return err
	}

	id := args[0]
	endpoint := fmt.Sprintf("%s/api/messages/%s/processed", strings.TrimRight(server, "/"), id)

	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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

	fmt.Printf("Message %s marked as unprocessed.\n", id)
	return nil
}

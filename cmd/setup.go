package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/lobsterclaw/pidge/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive config setup",
	Long:  "Create or update the pidge configuration file interactively.",
	RunE:  runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("pidge setup")
	fmt.Println()

	url := prompt(reader, "Gateway URL", "http://192.168.1.1:8080")
	user := prompt(reader, "Username", "admin")
	pass := prompt(reader, "Password", "")

	fmt.Println()
	fmt.Print("Testing connection... ")

	sdkConfig := smsgateway.Config{}.
		WithBaseURL(url).
		WithBasicAuth(user, pass)
	c := smsgateway.NewClient(sdkConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := c.CheckHealth(ctx)
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not connect to gateway: %w", err)
	}
	fmt.Printf("OK (version %s, status %s)\n", health.Version, health.Status)

	path := configPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return err
		}
		path = p
	}

	c2 := &config.Config{
		Gateway: config.GatewayConfig{
			URL:      url,
			Username: user,
			Password: pass,
		},
	}
	if err := c2.Save(path); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", path)
	return nil
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultVal
	}
	return text
}

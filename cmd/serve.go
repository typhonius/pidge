package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/lobsterclaw/pidge/internal/server"
	"github.com/lobsterclaw/pidge/internal/store"
	"github.com/spf13/cobra"
)

var (
	serveListen string
	serveDB     string
)

func init() {
	serveCmd.Flags().StringVar(&serveListen, "listen", "", "listen address (default from config or :3851)")
	serveCmd.Flags().StringVar(&serveDB, "db", "", "database path (default from config)")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start webhook receiver and REST API server",
	Long:  "Start a long-running server that receives SMS webhooks from the gateway and exposes a REST API.",
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	// Apply flag overrides
	listen := cfg.Server.Listen
	if serveListen != "" {
		listen = serveListen
	}
	dbPath := cfg.ExpandDBPath()
	if serveDB != "" {
		dbPath = serveDB
	}

	slog.Info("opening database", "path", dbPath)
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer st.Close()

	srv := server.New(st, client, cfg.Server.WebhookSecret)

	// Auto-register webhook
	if cfg.Server.AutoRegister && cfg.Server.WebhookURL != "" {
		if err := autoRegisterWebhook(cfg.Server.WebhookURL); err != nil {
			slog.Warn("auto-register webhook failed", "error", err)
		}
	}

	// Resolve TLS paths
	certFile := expandHome(cfg.Server.TLSCert)
	keyFile := expandHome(cfg.Server.TLSKey)

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(listen, certFile, keyFile)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		slog.Info("received signal", "signal", sig)
		if err := srv.Shutdown(10 * time.Second); err != nil {
			slog.Error("shutdown error", "error", err)
		}
		slog.Info("server stopped")
		return nil
	}
}

// autoRegisterWebhook checks existing webhooks and registers one for sms:received
// if none exists for our URL.
func autoRegisterWebhook(webhookURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hooks, err := client.ListWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("listing webhooks: %w", err)
	}

	// Check if our URL is already registered for sms:received
	for _, h := range hooks {
		if h.URL == webhookURL && h.Event == "sms:received" {
			slog.Info("webhook already registered", "id", h.ID, "url", h.URL)
			return nil
		}
	}

	hook, err := client.RegisterWebhook(ctx, smsgateway.Webhook{
		URL:   webhookURL,
		Event: "sms:received",
	})
	if err != nil {
		return fmt.Errorf("registering webhook: %w", err)
	}

	slog.Info("webhook registered", "id", hook.ID, "url", hook.URL, "event", hook.Event)
	return nil
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return home + path[1:]
		}
	}
	return path
}

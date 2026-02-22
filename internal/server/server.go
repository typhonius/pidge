package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/typhonius/pidge/internal/store"
)

// Server is the pidge HTTP server handling webhooks and the REST API.
type Server struct {
	store         *store.Store
	client        *smsgateway.Client
	webhookSecret string
	httpServer    *http.Server
}

// New creates a new Server.
func New(st *store.Store, client *smsgateway.Client, webhookSecret string) *Server {
	return &Server{
		store:         st,
		client:        client,
		webhookSecret: webhookSecret,
	}
}

// Start begins listening on the given address. If certFile and keyFile are
// non-empty, it serves HTTPS; otherwise plain HTTP.
func (s *Server) Start(addr, certFile, keyFile string) error {
	mux := http.NewServeMux()

	// Webhook endpoints (gateway POSTs here)
	mux.HandleFunc("POST /{$}", s.handleWebhook)
	mux.HandleFunc("POST /webhook", s.handleWebhook)

	// REST API
	mux.HandleFunc("GET /api/messages", s.handleListMessages)
	mux.HandleFunc("GET /api/messages/{id}", s.handleGetMessage)
	mux.HandleFunc("POST /api/messages/{id}/processed", s.handleMarkProcessed)
	mux.HandleFunc("DELETE /api/messages/{id}/processed", s.handleMarkUnprocessed)
	mux.HandleFunc("POST /api/messages/processed", s.handleMarkAllProcessed)
	mux.HandleFunc("POST /api/send", s.handleSend)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if certFile != "" && keyFile != "" {
		slog.Info("server starting (TLS)", "addr", addr)
		err := s.httpServer.ListenAndServeTLS(certFile, keyFile)
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}

	slog.Info("server starting", "addr", addr)
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server with a timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	slog.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}

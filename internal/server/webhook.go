package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/lobsterclaw/pidge/internal/store"
)

// webhookPayload represents the gateway's webhook POST body for sms:received.
type webhookPayload struct {
	DeviceID  string `json:"deviceId"`
	Event     string `json:"event"`
	ID        string `json:"id"`
	WebhookID string `json:"webhookId"`
	Payload   struct {
		MessageID   string `json:"messageId"`
		Message     string `json:"message"`
		PhoneNumber string `json:"phoneNumber"`
		SimNumber   int    `json:"simNumber"`
		ReceivedAt  string `json:"receivedAt"`
	} `json:"payload"`
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB max
	if err != nil {
		slog.Error("reading webhook body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if s.webhookSecret != "" {
		if !s.verifySignature(body, r.Header.Get("X-Signature"), r.Header.Get("X-Timestamp")) {
			slog.Warn("webhook signature verification failed")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("decoding webhook payload", "error", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if payload.Event != "sms:received" {
		slog.Debug("ignoring non-sms event", "event", payload.Event)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ignored","event":%q}`, payload.Event)
		return
	}

	if payload.ID == "" || payload.Payload.PhoneNumber == "" {
		slog.Warn("webhook missing required fields", "id", payload.ID)
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	receivedAt, err := time.Parse(time.RFC3339, payload.Payload.ReceivedAt)
	if err != nil {
		// Try alternate format
		receivedAt, err = time.Parse("2006-01-02T15:04:05.000-07:00", payload.Payload.ReceivedAt)
		if err != nil {
			receivedAt = time.Now()
		}
	}

	simNum := payload.Payload.SimNumber
	if simNum == 0 {
		simNum = 1
	}

	msg := store.ReceivedMessage{
		EventID:     payload.ID,
		MessageID:   payload.Payload.MessageID,
		DeviceID:    payload.DeviceID,
		PhoneNumber: payload.Payload.PhoneNumber,
		Message:     payload.Payload.Message,
		SimNumber:   simNum,
		ReceivedAt:  receivedAt,
	}

	if err := s.store.SaveMessage(msg); err != nil {
		slog.Error("saving message", "error", err, "event_id", payload.ID)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	slog.Info("message received",
		"event_id", payload.ID,
		"from", payload.Payload.PhoneNumber,
		"preview", truncate(payload.Payload.Message, 40),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"stored","eventId":%q}`, payload.ID)
}

func (s *Server) verifySignature(body []byte, signature, timestamp string) bool {
	if signature == "" || timestamp == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

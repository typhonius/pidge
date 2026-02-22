package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/typhonius/pidge/internal/store"
)

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.ListFilter{
		Phone: q.Get("phone"),
	}

	if v := q.Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Since = &t
		}
	}
	if v := q.Get("before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Before = &t
		}
	}
	if v := q.Get("processed"); v != "" {
		b := v == "true" || v == "1"
		f.Processed = &b
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}

	messages, err := s.store.ListMessages(f)
	if err != nil {
		slog.Error("listing messages", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	if messages == nil {
		messages = []store.ReceivedMessage{}
	}
	writeJSON(w, http.StatusOK, messages)
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	msg, err := s.store.GetMessage(id)
	if err != nil {
		slog.Error("getting message", "error", err, "id", id)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if msg == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleMarkProcessed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.store.MarkProcessed(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		slog.Error("marking processed", "error", err, "id", id)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// sendRequest is the JSON body for POST /api/send.
type sendRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Message     string `json:"message"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.PhoneNumber == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phoneNumber and message are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	state, err := s.client.Send(ctx, smsgateway.Message{
		TextMessage:  &smsgateway.TextMessage{Text: req.Message},
		PhoneNumbers: []string{req.PhoneNumber},
	})
	if err != nil {
		slog.Error("sending SMS", "error", err, "to", req.PhoneNumber)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("gateway error: %v", err)})
		return
	}

	slog.Info("SMS sent", "id", state.ID, "to", req.PhoneNumber)
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{
		"status": "ok",
		"server": "running",
	}

	// Store stats
	stats, err := s.store.Stats()
	if err != nil {
		result["store"] = map[string]string{"error": err.Error()}
	} else {
		result["store"] = stats
	}

	// Gateway health
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	health, err := s.client.CheckHealth(ctx)
	if err != nil {
		result["gateway"] = map[string]string{"status": "unreachable", "error": err.Error()}
	} else {
		result["gateway"] = map[string]any{
			"status":  health.Status,
			"version": health.Version,
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

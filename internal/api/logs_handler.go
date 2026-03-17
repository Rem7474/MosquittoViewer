package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
)

func (s *Server) GetLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UsernameFromContext(r.Context()); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	limit := parseInt(q.Get("limit"), 100)
	if limit <= 0 {
		limit = 100
	}
	offset := parseInt(q.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	filters := logwatcher.Filters{
		Level:    q.Get("level"),
		Query:    q.Get("q"),
		ClientID: q.Get("clientId"),
		Topic:    q.Get("topic"),
	}
	if from := q.Get("from"); from != "" {
		if parsed, err := time.Parse(time.RFC3339, from); err == nil {
			filters.From = &parsed
		}
	}
	if to := q.Get("to"); to != "" {
		if parsed, err := time.Parse(time.RFC3339, to); err == nil {
			filters.To = &parsed
		}
	}

	all := s.watcher.Recent(0, filters)
	total := len(all)
	if offset >= total {
		writeJSON(w, http.StatusOK, map[string]any{"data": []logwatcher.LogEntry{}, "total": len(all)})
		return
	}

	// Tail semantics: return most recent entries first based on limit/offset.
	end := total - offset
	start := end - limit
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	window := all[start:end]

	// API returns newest first.
	for i, j := 0, len(window)-1; i < j; i, j = i+1, j-1 {
		window[i], window[j] = window[j], window[i]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  window,
		"total": len(all),
	})
}

func parseInt(v string, fallback int) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

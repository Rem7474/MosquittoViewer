package api

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
)

// GetSources returns the list of configured log sources.
func (s *Server) GetSources(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UsernameFromContext(r.Context()); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	type SourceInfo struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	out := make([]SourceInfo, 0, len(s.sourceOrder))
	for _, name := range s.sourceOrder {
		if w, ok := s.watchers[name]; ok {
			out = append(out, SourceInfo{Name: name, Path: w.Path()})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": out})
}

// GetLogs returns buffered log entries, optionally filtered by source.
func (s *Server) GetLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UsernameFromContext(r.Context()); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	limit := parseInt(q.Get("limit"), 200)
	if limit <= 0 {
		limit = 200
	}
	offset := parseInt(q.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	source := q.Get("source") // "" = all sources

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

	// Collect entries from the requested source(s).
	var all []logwatcher.LogEntry
	if source != "" {
		if watcher, ok := s.watchers[source]; ok {
			all = watcher.Recent(0, filters)
		}
	} else {
		for _, name := range s.sourceOrder {
			all = append(all, s.watchers[name].Recent(0, filters)...)
		}
		// Sort oldest→newest so pagination is deterministic.
		sort.Slice(all, func(i, j int) bool {
			return all[i].Timestamp.Before(all[j].Timestamp)
		})
	}

	total := len(all)
	if offset >= total {
		writeJSON(w, http.StatusOK, map[string]any{"data": []logwatcher.LogEntry{}, "total": total})
		return
	}

	// Tail semantics: return most recent entries based on limit/offset.
	end := total - offset
	start := end - limit
	if start < 0 {
		start = 0
	}
	window := all[start:end]

	// Return newest first.
	for i, j := 0, len(window)-1; i < j; i, j = i+1, j-1 {
		window[i], window[j] = window[j], window[i]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  window,
		"total": total,
	})
}

func parseInt(v string, fallback int) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

package logwatcher

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	Path        string
	Format      string
	CustomRegex string
	BufferSize  int
	Debug       bool
}

type Watcher struct {
	entries []LogEntry
	head    int
	count   int
	mu      sync.RWMutex

	outCh chan LogEntry
	subs  map[chan LogEntry]struct{}

	config     Config
	parser     *Parser
	file       *os.File
	lastOffset int64
	nextID     atomic.Int64
	path       string
	subMu      sync.RWMutex
}

func New(cfg Config) *Watcher {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	parser, _ := NewParser(cfg.Format, cfg.CustomRegex)
	if parser == nil {
		parser, _ = NewParser("mosquitto_standard", "")
	}
	return &Watcher{
		entries: make([]LogEntry, cfg.BufferSize),
		outCh:   make(chan LogEntry, cfg.BufferSize),
		subs:    make(map[chan LogEntry]struct{}),
		config:  cfg,
		parser:  parser,
		path:    cfg.Path,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	if strings.TrimSpace(w.path) == "" {
		return errors.New("log path is empty")
	}
	if err := w.openCurrentFile(true); err != nil && !os.IsNotExist(err) {
		return err
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fsw.Close()

	if err := fsw.Add(filepath.Dir(w.path)); err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.closeFile()
			return nil
		case ev := <-fsw.Events:
			if filepath.Clean(ev.Name) != filepath.Clean(w.path) {
				continue
			}
			switch {
			case ev.Op&(fsnotify.Write|fsnotify.Chmod) != 0:
				_ = w.readNewLines()
			case ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0:
				w.closeFile()
				w.lastOffset = 0
			case ev.Op&fsnotify.Create != 0:
				_ = w.openCurrentFile(false)
				_ = w.readNewLines()
			}
		case <-ticker.C:
			_ = w.ensureOpen()
			_ = w.readNewLines()
		case <-fsw.Errors:
			// Keep watching despite transient fsnotify errors.
		}
	}
}

func (w *Watcher) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 256)
	w.subMu.Lock()
	w.subs[ch] = struct{}{}
	w.subMu.Unlock()
	return ch
}

func (w *Watcher) Recent(n int, filters Filters) []LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.count == 0 {
		return nil
	}
	if n <= 0 || n > w.count {
		n = w.count
	}

	start := (w.head - w.count + len(w.entries)) % len(w.entries)
	out := make([]LogEntry, 0, n)
	for i := 0; i < w.count; i++ {
		idx := (start + i) % len(w.entries)
		entry := w.entries[idx]
		if !matchFilters(entry, filters) {
			continue
		}
		out = append(out, entry)
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}

func (w *Watcher) ensureOpen() error {
	if w.file != nil {
		return nil
	}
	return w.openCurrentFile(false)
}

func (w *Watcher) openCurrentFile(tailToEnd bool) error {
	f, err := os.Open(w.path)
	if err != nil {
		return err
	}
	w.file = f
	if tailToEnd {
		off, err := f.Seek(0, 2)
		if err == nil {
			w.lastOffset = off
		}
	}
	return nil
}

func (w *Watcher) closeFile() {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
}

func (w *Watcher) readNewLines() error {
	if w.file == nil {
		return nil
	}
	if _, err := w.file.Seek(w.lastOffset, 0); err != nil {
		return err
	}

	scanner := bufio.NewScanner(w.file)
	for scanner.Scan() {
		line := scanner.Text()
		id := w.nextID.Add(1)
		entry, err := w.parser.ParseLine(line, id)
		if err != nil {
			if w.config.Debug {
				entry = LogEntry{ID: id, Timestamp: time.Now().UTC(), Level: "DEBUG", Message: line, Raw: line}
			} else {
				continue
			}
		}
		w.append(entry)
		w.publish(entry)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	off, err := w.file.Seek(0, 1)
	if err == nil {
		w.lastOffset = off
	}
	return nil
}

func (w *Watcher) append(entry LogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries[w.head] = entry
	w.head = (w.head + 1) % len(w.entries)
	if w.count < len(w.entries) {
		w.count++
	}
}

func (w *Watcher) publish(entry LogEntry) {
	select {
	case w.outCh <- entry:
	default:
	}

	w.subMu.RLock()
	for ch := range w.subs {
		select {
		case ch <- entry:
		default:
		}
	}
	w.subMu.RUnlock()
}

func matchFilters(e LogEntry, f Filters) bool {
	if f.Level != "" && strings.ToUpper(f.Level) != "ALL" && strings.ToUpper(e.Level) != strings.ToUpper(f.Level) {
		return false
	}
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		if !strings.Contains(strings.ToLower(e.Message), q) && !strings.Contains(strings.ToLower(e.Raw), q) {
			return false
		}
	}
	if f.ClientID != "" && !strings.EqualFold(e.ClientID, f.ClientID) {
		return false
	}
	if f.Topic != "" && !strings.Contains(strings.ToLower(e.Topic), strings.ToLower(f.Topic)) {
		return false
	}
	if f.From != nil && e.Timestamp.Before(*f.From) {
		return false
	}
	if f.To != nil && e.Timestamp.After(*f.To) {
		return false
	}
	return true
}

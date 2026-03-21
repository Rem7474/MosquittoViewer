package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mosquittoviewer "github.com/example/mosquitto-viewer"
	"github.com/example/mosquitto-viewer/internal/api"
	"github.com/example/mosquitto-viewer/internal/config"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
	"github.com/example/mosquitto-viewer/internal/ws"
)

func main() {
	configPath := flag.String("config", "./configs/config.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if len(cfg.Logs) == 0 {
		logger.Error("no log sources configured – add at least one entry under 'logs:' in config")
		os.Exit(1)
	}

	hub := ws.NewHub()
	go hub.Run()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchers := make(map[string]*logwatcher.Watcher, len(cfg.Logs))
	sourceOrder := make([]string, 0, len(cfg.Logs))

	for _, src := range cfg.Logs {
		w := logwatcher.New(logwatcher.Config{
			Name:                src.Name,
			Path:                src.Path,
			Format:              src.Format,
			CustomRegex:         src.CustomRegex,
			BufferSize:          src.BufferSize,
			ReadExistingOnStart: src.ReadExistingOnStart,
			Debug:               cfg.Debug,
		})
		watchers[src.Name] = w
		sourceOrder = append(sourceOrder, src.Name)

		// Start watcher in background.
		go func(name string, watcher *logwatcher.Watcher) {
			if err := watcher.Start(ctx); err != nil {
				logger.Error("watcher stopped", "source", name, "error", err)
			}
		}(src.Name, w)

		// Forward entries to the hub.
		sub := w.Subscribe()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case e := <-sub:
					hub.Broadcast(e)
				}
			}
		}()
	}

	router := api.NewRouter(api.Options{
		JWTConfig:    cfg.Auth.JWT,
		Users:        cfg.Auth.Users,
		Watchers:     watchers,
		SourceOrder:  sourceOrder,
		Sources:      cfg.Logs,
		Hub:          hub,
		WebFS:        mosquittoviewer.WebFS,
		AllowDevCORS: true,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

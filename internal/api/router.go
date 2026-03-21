package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/config"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
	"github.com/example/mosquitto-viewer/internal/ws"
)

type Server struct {
	jwtCfg      config.JWTConfig
	users       map[string]string
	watchers    map[string]*logwatcher.Watcher // keyed by source name
	sourceOrder []string                        // ordered list of source names
	sources     []config.LogSourceConfig        // raw config, for metadata
	hub         *ws.Hub
	webFS       fs.FS
	allowDevCORS bool
}

type Options struct {
	JWTConfig    config.JWTConfig
	Users        []config.UserConfig
	Watchers     map[string]*logwatcher.Watcher
	SourceOrder  []string
	Sources      []config.LogSourceConfig
	Hub          *ws.Hub
	WebFS        fs.FS
	AllowDevCORS bool
}

func NewRouter(opts Options) http.Handler {
	users := make(map[string]string, len(opts.Users))
	for _, u := range opts.Users {
		users[u.Username] = u.PasswordHash
	}

	s := &Server{
		jwtCfg:       opts.JWTConfig,
		users:        users,
		watchers:     opts.Watchers,
		sourceOrder:  opts.SourceOrder,
		sources:      opts.Sources,
		hub:          opts.Hub,
		webFS:        opts.WebFS,
		allowDevCORS: opts.AllowDevCORS,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", requireMethods([]string{http.MethodPost}, s.Login))
	mux.HandleFunc("/api/auth/refresh", requireMethods([]string{http.MethodPost}, s.Refresh))
	mux.Handle("/api/logs", auth.Middleware(s.jwtCfg, requireMethodsHandler([]string{http.MethodGet}, http.HandlerFunc(s.GetLogs))))
	mux.Handle("/api/sources", auth.Middleware(s.jwtCfg, requireMethodsHandler([]string{http.MethodGet}, http.HandlerFunc(s.GetSources))))
	mux.HandleFunc("/api/ws", requireMethods([]string{http.MethodGet}, s.ServeWS))
	mux.HandleFunc("/api/health", requireMethods([]string{http.MethodGet}, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0.0"})
	}))

	static := http.FileServer(http.FS(opts.WebFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Static assets have a file extension (e.g. /assets/main.js, /favicon.ico).
		// SPA routes (/login, /dashboard, …) have no extension and must all receive
		// index.html so Vue Router can handle them client-side.
		if r.URL.Path != "/" && strings.Contains(path.Base(r.URL.Path), ".") {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/web" + r.URL.Path
			static.ServeHTTP(w, r2)
			return
		}
		index, err := fs.ReadFile(opts.WebFS, "web/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	})

	h := http.Handler(mux)
	if s.allowDevCORS {
		h = withCORS(h)
	}
	return h
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func requireMethods(allowed []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, m := range allowed {
			if r.Method == m {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func requireMethodsHandler(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, m := range allowed {
			if r.Method == m {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

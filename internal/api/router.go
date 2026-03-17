package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/config"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
	"github.com/example/mosquitto-viewer/internal/ws"
)

type Server struct {
	jwtCfg         config.JWTConfig
	users          map[string]string
	watcher        *logwatcher.Watcher
	hub            *ws.Hub
	webFS          fs.FS
	allowDevCORS   bool
	defaultFilters logwatcher.Filters
}

type Options struct {
	JWTConfig     config.JWTConfig
	Users         []config.UserConfig
	Watcher       *logwatcher.Watcher
	Hub           *ws.Hub
	WebFS         fs.FS
	AllowDevCORS  bool
	DefaultFilter logwatcher.Filters
}

func NewRouter(opts Options) http.Handler {
	users := make(map[string]string, len(opts.Users))
	for _, u := range opts.Users {
		users[u.Username] = u.PasswordHash
	}

	s := &Server{
		jwtCfg:         opts.JWTConfig,
		users:          users,
		watcher:        opts.Watcher,
		hub:            opts.Hub,
		webFS:          opts.WebFS,
		allowDevCORS:   opts.AllowDevCORS,
		defaultFilters: opts.DefaultFilter,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", s.Login)
	mux.HandleFunc("POST /api/auth/refresh", s.Refresh)
	mux.Handle("GET /api/logs", auth.Middleware(s.jwtCfg, http.HandlerFunc(s.GetLogs)))
	mux.HandleFunc("GET /api/ws", s.ServeWS)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0.0"})
	})

	static := http.FileServer(http.FS(opts.WebFS))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		if r.URL.Path == "/" {
			r2.URL.Path = "/web/index.html"
		} else {
			r2.URL.Path = "/web" + r.URL.Path
		}
		static.ServeHTTP(w, r2)
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

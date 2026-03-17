package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/mosquitto-viewer/internal/config"
)

type contextKey string

const UsernameContextKey contextKey = "username"

func Middleware(cfg config.JWTConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		username, err := ValidateAccessToken(token, cfg)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), UsernameContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(UsernameContextKey)
	username, ok := v.(string)
	return username, ok
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

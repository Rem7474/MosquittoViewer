package api

import (
	"encoding/json"
	"net/http"

	"github.com/example/mosquitto-viewer/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	hash, ok := s.users[req.Username]
	if !ok || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	access, refresh, err := auth.GenerateTokenPair(req.Username, s.jwtCfg)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
		"username":      req.Username,
	})
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	username, err := auth.ValidateRefreshToken(req.RefreshToken, s.jwtCfg)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	access, refresh, err := auth.GenerateTokenPair(username, s.jwtCfg)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
		"username":      username,
	})
}

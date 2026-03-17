package api

import (
	"encoding/json"
	"net/http"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/ws"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	username, err := auth.ValidateAccessToken(token, s.jwtCfg)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &ws.Client{Conn: conn, Send: make(chan []byte, 256), UserID: username}
	s.hub.Register(client)

	for _, e := range s.watcher.Recent(100, s.defaultFilters) {
		if b, err := json.Marshal(e); err == nil {
			select {
			case client.Send <- b:
			default:
			}
		}
	}

	go client.WritePump(func() { s.hub.Unregister(client) })
	go client.ReadPump(func() { s.hub.Unregister(client) })
}

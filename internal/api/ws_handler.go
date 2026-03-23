package api

import (
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

	client := &ws.Client{Conn: conn, Send: make(chan []byte, 512), UserID: username}
	s.hub.Register(client)

	// History is loaded by the client via GET /api/logs.
	// The WebSocket only delivers entries that arrive after this connection is established.

	go client.WritePump(func() { s.hub.Unregister(client) })
	go client.ReadPump(func() { s.hub.Unregister(client) })
}

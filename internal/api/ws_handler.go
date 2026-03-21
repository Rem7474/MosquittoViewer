package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/example/mosquitto-viewer/internal/auth"
	"github.com/example/mosquitto-viewer/internal/logwatcher"
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

	// Send recent entries from all sources, merged and sorted oldest→newest,
	// so the client sees a coherent history on connect.
	var recent []logwatcher.LogEntry
	for _, name := range s.sourceOrder {
		recent = append(recent, s.watchers[name].Recent(200, logwatcher.Filters{})...)
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Timestamp.Before(recent[j].Timestamp)
	})
	// Keep last 200 across all sources.
	if len(recent) > 200 {
		recent = recent[len(recent)-200:]
	}
	for _, e := range recent {
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

package ws

import (
	"encoding/json"
	"time"

	"github.com/example/mosquitto-viewer/internal/logwatcher"
	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
	pingEvery = 30 * time.Second
)

type Client struct {
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if h.clients[c] {
				delete(h.clients, c)
				close(c.Send)
				_ = c.Conn.Close()
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.Send <- msg:
				default:
					// Silent drop: client is too slow.
				}
			}
		}
	}
}

func (h *Hub) Broadcast(entry logwatcher.LogEntry) {
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- b:
	default:
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (c *Client) ReadPump(onClose func()) {
	defer onClose()
	c.Conn.SetReadLimit(2048)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) WritePump(onClose func()) {
	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	defer onClose()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

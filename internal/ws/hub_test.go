package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/example/mosquitto-viewer/internal/logwatcher"
)

func TestBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{Send: make(chan []byte, 256)}
	c2 := &Client{Send: make(chan []byte, 256)}
	hub.Register(c1)
	hub.Register(c2)

	entry := logwatcher.LogEntry{ID: 1, Level: "INFO", Message: "hello"}
	hub.Broadcast(entry)

	assertMessage(t, c1.Send)
	assertMessage(t, c2.Send)
}

func TestClientDropOnFull(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{Send: make(chan []byte, 1)}
	hub.Register(client)

	client.Send <- []byte(`{"stale":true}`)
	hub.Broadcast(logwatcher.LogEntry{ID: 2, Level: "INFO", Message: "drop"})

	select {
	case <-client.Send:
		// one stale message consumed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected stale message to remain in full buffer")
	}

	select {
	case <-client.Send:
		t.Fatal("expected no second message because hub should drop on full buffer")
	default:
	}
}

func assertMessage(t *testing.T, ch <-chan []byte) {
	t.Helper()
	select {
	case msg := <-ch:
		payload := logwatcher.LogEntry{}
		if err := json.Unmarshal(msg, &payload); err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting broadcast")
	}
}

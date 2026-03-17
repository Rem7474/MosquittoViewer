package logwatcher

import "testing"

func TestParseStandardLine(t *testing.T) {
	p, err := NewParser("mosquitto_standard", "")
	if err != nil {
		t.Fatal(err)
	}

	e, err := p.ParseLine("1710000001: New client connected from 192.168.1.10 as my-client (p2, c1, k60).", 1)
	if err != nil {
		t.Fatal(err)
	}

	if e.Level != "INFO" {
		t.Fatalf("expected INFO, got %s", e.Level)
	}
	if e.ClientID != "my-client" {
		t.Fatalf("expected client id my-client, got %s", e.ClientID)
	}
}

func TestParseCustomRegex(t *testing.T) {
	p, err := NewParser("custom", `^(?P<ts>\d+): \[(?P<level>\w+)\] \[(?P<plugin>\w+)\] (?P<msg>.+)$`)
	if err != nil {
		t.Fatal(err)
	}

	e, err := p.ParseLine("1710000003: [WARN] [acl] topic /a/b denied", 7)
	if err != nil {
		t.Fatal(err)
	}

	if e.Level != "WARN" {
		t.Fatalf("expected WARN, got %s", e.Level)
	}
	if e.Plugin != "acl" {
		t.Fatalf("expected plugin acl, got %s", e.Plugin)
	}
}

func TestExtractClientID(t *testing.T) {
	got := extractClientID("New client connected from 10.0.0.2 as sensor-42 (p2, c1, k60).")
	if got != "sensor-42" {
		t.Fatalf("expected sensor-42, got %s", got)
	}
}

func TestLevelDetection(t *testing.T) {
	cases := map[string]string{
		"AUTH failed for user":       "ERROR",
		"connection timeout warning": "WARN",
		"debug packet received":      "DEBUG",
		"New client connected":       "INFO",
	}

	for message, expected := range cases {
		if got := detectLevel(message); got != expected {
			t.Fatalf("message %q expected %s got %s", message, expected, got)
		}
	}
}

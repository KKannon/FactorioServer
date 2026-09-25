package factorio

import (
	"testing"
	"time"
)

func TestObservePlayerRuntimeEvents(t *testing.T) {
	server := &Server{onlinePlayers: make(map[string]time.Time), startedAt: time.Now().Add(-time.Minute), processID: 42}
	server.observePlayerEvent("12.000 [JOIN] Alice joined the game")
	server.observePlayerEvent("13.000 [JOIN] Bob joined the game")
	server.observePlayerEvent("14.000 [LEAVE] Alice left the game")
	metrics := server.RuntimeMetrics()
	if metrics.ProcessID != 42 || metrics.OnlineCount != 1 || metrics.OnlinePlayers[0].Name != "Bob" {
		t.Fatalf("unexpected runtime metrics: %#v", metrics)
	}
	if metrics.UptimeSeconds < 59 {
		t.Fatalf("expected server uptime, got %d", metrics.UptimeSeconds)
	}
}

func TestObservePlayerRuntimeEventsIgnoresMalformedLines(t *testing.T) {
	server := &Server{}
	server.observePlayerEvent("ordinary Factorio log line")
	server.observePlayerEvent("[JOIN]  joined the game")
	if metrics := server.RuntimeMetrics(); metrics.OnlineCount != 0 {
		t.Fatalf("unexpected players: %#v", metrics.OnlinePlayers)
	}
}

package factorio

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

var playerRuntimeEventPattern = regexp.MustCompile(`\[(JOIN|LEAVE)\]\s+(.+?)\s+(?:joined|left) the game(?:\s|$)`)

type RuntimePlayer struct {
	Name             string `json:"name"`
	ConnectedSeconds int64  `json:"connected_seconds"`
}

type RuntimeMetrics struct {
	ProcessID        int             `json:"process_id"`
	UptimeSeconds    int64           `json:"uptime_seconds"`
	OnlinePlayers    []RuntimePlayer `json:"online_players"`
	OnlineCount      int             `json:"online_count"`
	PlayerDataSource string          `json:"player_data_source"`
}

func (server *Server) observePlayerEvent(line string) {
	match := playerRuntimeEventPattern.FindStringSubmatch(line)
	if len(match) != 3 {
		return
	}
	name := strings.TrimSpace(match[2])
	if name == "" || len(name) > 64 || strings.ContainsAny(name, "\r\n") {
		return
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.onlinePlayers == nil {
		server.onlinePlayers = make(map[string]time.Time)
	}
	if match[1] == "JOIN" {
		if _, exists := server.onlinePlayers[name]; !exists {
			server.onlinePlayers[name] = time.Now()
		}
	} else {
		delete(server.onlinePlayers, name)
	}
}

func (server *Server) RuntimeMetrics() RuntimeMetrics {
	server.mu.RLock()
	defer server.mu.RUnlock()
	now := time.Now()
	metrics := RuntimeMetrics{
		ProcessID:        server.processID,
		OnlinePlayers:    make([]RuntimePlayer, 0, len(server.onlinePlayers)),
		PlayerDataSource: "factorio-log-events",
	}
	if !server.startedAt.IsZero() {
		metrics.UptimeSeconds = int64(now.Sub(server.startedAt).Seconds())
	}
	for name, connectedAt := range server.onlinePlayers {
		metrics.OnlinePlayers = append(metrics.OnlinePlayers, RuntimePlayer{
			Name: name, ConnectedSeconds: int64(now.Sub(connectedAt).Seconds()),
		})
	}
	sort.Slice(metrics.OnlinePlayers, func(i, j int) bool { return metrics.OnlinePlayers[i].Name < metrics.OnlinePlayers[j].Name })
	metrics.OnlineCount = len(metrics.OnlinePlayers)
	return metrics
}

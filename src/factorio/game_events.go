package factorio

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const gameEventsResponse = "FSM_GAME_EVENTS:"

type GameEvent struct {
	ID      int64                  `json:"id"`
	Event   string                 `json:"event"`
	Tick    int64                  `json:"tick"`
	Payload map[string]interface{} `json:"payload"`
}

type bridgeEventsResponse struct {
	SchemaVersion int                      `json:"schema_version"`
	LatestID      int64                    `json:"latest_id"`
	Events        []map[string]interface{} `json:"events"`
}

func parseBridgeEvents(response string) (bridgeEventsResponse, error) {
	var result bridgeEventsResponse
	index := strings.Index(response, gameEventsResponse)
	if index < 0 {
		return result, errors.New("game event bridge did not answer")
	}
	payload := strings.TrimSpace(response[index+len(gameEventsResponse):])
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return result, fmt.Errorf("decode game events: %w", err)
	}
	if result.SchemaVersion != 1 {
		return result, errors.New("unsupported game event response")
	}
	return result, nil
}

func ReadBridgeEvents(afterID int64) ([]GameEvent, int64, error) {
	response, err := requestBridgeCommand("/fsm-events " + strconv.FormatInt(afterID, 10))
	if err != nil {
		return nil, afterID, err
	}
	decoded, err := parseBridgeEvents(response)
	if err != nil {
		return nil, afterID, err
	}
	events := make([]GameEvent, 0, len(decoded.Events))
	for _, raw := range decoded.Events {
		id, _ := raw["id"].(float64)
		tick, _ := raw["tick"].(float64)
		kind, _ := raw["event"].(string)
		delete(raw, "id")
		delete(raw, "tick")
		delete(raw, "event")
		if int64(id) <= afterID || kind == "" {
			continue
		}
		events = append(events, GameEvent{ID: int64(id), Tick: int64(tick), Event: kind, Payload: raw})
	}
	return events, decoded.LatestID, nil
}

func ConfigureBridgeEvents(eventTypes []string) error {
	types := append([]string(nil), eventTypes...)
	sort.Strings(types)
	response, err := requestBridgeCommand("/fsm-events-config " + strings.Join(types, ","))
	if err != nil {
		return err
	}
	if !strings.Contains(response, gameEventsResponse) {
		return errors.New("game event bridge did not accept the configuration")
	}
	return nil
}

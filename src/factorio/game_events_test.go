package factorio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBridgeEvents(t *testing.T) {
	response := "command output\n" + gameEventsResponse + `{"schema_version":1,"latest_id":7,"events":[{"id":7,"event":"on_entity_damaged","tick":123,"final_damage":8.5}]}`
	decoded, err := parseBridgeEvents(response)
	require.NoError(t, err)
	require.Equal(t, int64(7), decoded.LatestID)
	require.Len(t, decoded.Events, 1)
	require.Equal(t, "on_entity_damaged", decoded.Events[0]["event"])
}

func TestParseBridgeEventsRejectsMissingPrefix(t *testing.T) {
	_, err := parseBridgeEvents(`{"schema_version":1}`)
	require.Error(t, err)
}

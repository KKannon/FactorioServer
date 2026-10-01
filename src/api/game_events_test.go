package api

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderEventTemplateUsesKnownNestedValues(t *testing.T) {
	payload := map[string]interface{}{"event": "on_player_died", "player": map[string]interface{}{"name": "Engineer"}}
	require.Equal(t, "on_player_died: Engineer ()", renderEventTemplate("{{event}}: {{ player.name }} ({{missing}})", payload))
}

func TestValidateGameEventRuleEnforcesSafeCooldowns(t *testing.T) {
	webhook := GameEventRule{Name: "damage", Event: "on_entity_damaged", Action: "panel", Enabled: true}
	require.NoError(t, validateGameEventRule(&webhook))
	require.Equal(t, 5, webhook.CooldownSeconds)

	save := GameEventRule{Name: "save", Event: "on_player_joined_game", Action: "save_game", Enabled: true}
	require.NoError(t, validateGameEventRule(&save))
	require.Equal(t, 60, save.CooldownSeconds)
}

func TestPublicAddressRejectsInternalTargets(t *testing.T) {
	require.False(t, publicAddress(net.ParseIP("127.0.0.1")))
	require.False(t, publicAddress(net.ParseIP("10.1.2.3")))
	require.True(t, publicAddress(net.ParseIP("1.1.1.1")))
}

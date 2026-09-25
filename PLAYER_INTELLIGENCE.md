# Player intelligence

Factorio Server Manager does not parse the binary save format or fabricate
player data. Detailed profiles are provided by the optional bundled
`factorio-server-manager-bridge` mod through a read-only RCON command.

The bridge must be installed while Factorio is stopped and becomes active the
next time the world starts. Installing it changes the world's active mod set,
so the panel always requires an explicit management action and confirmation.

## Data sources and limitations

| Data | Source | Availability |
| --- | --- | --- |
| Online state | Native Factorio player object | Live and reliable |
| Total playtime in the world | `LuaPlayer.online_time` | All sessions in the current save |
| Position and surface | `LuaPlayer.physical_position` and `physical_surface` | Current player controller |
| Force and permission group | Native Lua player objects | Current world state |
| Health | Character entity | Available when the player has a character |
| Inventory, weapons, ammunition and armor | Native Lua inventories | Current world state |
| Armor equipment | Native equipment grid | Current world state |
| Crafting queue | Native Lua player control | Current queue |
| Deaths, crafted items and distance | Bridge event counters | Only from bridge installation onward |
| Per-player kills | — | Not reported because native kill statistics are force-level and attribution is ambiguous |
| Historical distance/deaths | — | Not reconstructed from the binary save |
| Unlocked achievements | — | Not exposed reliably per player by the runtime API |

Members can retrieve only the profile whose Factorio username exactly matches
their authenticated account. Support, manager and administrator roles can view
all profiles. Whitelist, administrator and ban controls remain restricted to
management roles.

The custom `/fsm-players` command accepts requests from RCON. If invoked by an
in-game player, it returns data only for an administrator. The bridge does not
mutate inventories, characters, permissions or world state beyond its own
forward-looking counters.

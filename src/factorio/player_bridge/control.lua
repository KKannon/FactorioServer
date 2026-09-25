local response_prefix = "FSM_PLAYER_INTELLIGENCE:"

local function bridge_stats(player_index)
    storage.fsm_player_statistics = storage.fsm_player_statistics or {}
    local current = storage.fsm_player_statistics[player_index]
    if not current then
        current = {deaths = 0, crafted_items = 0, distance = 0}
        storage.fsm_player_statistics[player_index] = current
    end
    return current
end

local function quality_name(value)
    if type(value) == "string" then return value end
    if value and value.name then return value.name end
    return "normal"
end

local function inventory_contents(inventory)
    local result = {}
    if not inventory or not inventory.valid then return result end
    for key, value in pairs(inventory.get_contents()) do
        if type(value) == "number" then
            table.insert(result, {name = key, quality = "normal", count = value})
        else
            table.insert(result, {
                name = value.name,
                quality = quality_name(value.quality),
                count = value.count
            })
        end
    end
    table.sort(result, function(left, right)
        if left.name == right.name then return left.quality < right.quality end
        return left.name < right.name
    end)
    return result
end

local function equipment_contents(player)
    local result = {}
    local armor = player.get_inventory(defines.inventory.character_armor)
    if not armor or not armor.valid then return result end
    for slot = 1, #armor do
        local stack = armor[slot]
        if stack and stack.valid_for_read and stack.grid then
            for _, equipment in pairs(stack.grid.equipment) do
                table.insert(result, {
                    name = equipment.name,
                    quality = quality_name(equipment.quality),
                    shield = equipment.shield or 0,
                    energy = equipment.energy or 0
                })
            end
        end
    end
    return result
end

local function crafting_queue(player)
    local result = {}
    if not player.crafting_queue then return result end
    for _, item in pairs(player.crafting_queue) do
        table.insert(result, {
            recipe = item.recipe,
            count = item.count,
            prerequisite = item.prerequisite == true
        })
    end
    return result
end

local function player_snapshot(player)
    local stats = bridge_stats(player.index)
    local position = player.physical_position
    local surface = player.physical_surface
    local character = player.character
    local health = nil
    local max_health = nil
    if character and character.valid then
        health = character.health
        max_health = character.prototype.max_health + player.character_health_bonus
    end
    return {
        name = player.name,
        connected = player.connected,
        admin = player.admin,
        force = player.force.name,
        permission_group = player.permission_group and player.permission_group.name or nil,
        surface = surface and surface.name or nil,
        position = position and {x = position.x, y = position.y} or nil,
        online_ticks = player.online_time,
        last_online_tick = player.last_online,
        afk_ticks = player.afk_time,
        health = health,
        max_health = max_health,
        inventory = inventory_contents(player.get_main_inventory()),
        guns = inventory_contents(player.get_inventory(defines.inventory.character_guns)),
        ammo = inventory_contents(player.get_inventory(defines.inventory.character_ammo)),
        armor = inventory_contents(player.get_inventory(defines.inventory.character_armor)),
        trash = inventory_contents(player.get_inventory(defines.inventory.character_trash)),
        equipment = equipment_contents(player),
        crafting_queue = crafting_queue(player),
        statistics = {
            scope = "since-bridge-installation",
            deaths = stats.deaths,
            crafted_items = stats.crafted_items,
            distance = stats.distance
        }
    }
end

local function snapshot()
    local players = {}
    for _, player in pairs(game.players) do
        table.insert(players, player_snapshot(player))
    end
    table.sort(players, function(left, right) return left.name < right.name end)
    return {
        schema_version = 1,
        generated_tick = game.tick,
        source = "factorio-server-manager-bridge",
        players = players,
        capabilities = {
            inventory = true,
            position = true,
            playtime = true,
            equipment = true,
            crafting_queue = true,
            health = true,
            tracked_statistics = true,
            achievements = false,
            achievements_reason = "The Factorio runtime API does not expose reliable per-player unlocked achievement state."
        }
    }
end

script.on_init(function()
    storage.fsm_player_statistics = storage.fsm_player_statistics or {}
    for _, player in pairs(game.players) do bridge_stats(player.index) end
end)

script.on_configuration_changed(function()
    storage.fsm_player_statistics = storage.fsm_player_statistics or {}
    for _, player in pairs(game.players) do bridge_stats(player.index) end
end)

script.on_event(defines.events.on_player_created, function(event)
    bridge_stats(event.player_index)
end)

script.on_event(defines.events.on_player_died, function(event)
    local stats = bridge_stats(event.player_index)
    stats.deaths = stats.deaths + 1
end)

script.on_event(defines.events.on_player_crafted_item, function(event)
    local stats = bridge_stats(event.player_index)
    stats.crafted_items = stats.crafted_items + (event.item_stack and event.item_stack.count or 1)
end)

script.on_event(defines.events.on_player_changed_position, function(event)
    local player = game.get_player(event.player_index)
    if not player then return end
    local stats = bridge_stats(event.player_index)
    local position = player.physical_position
    local surface = player.physical_surface
    if stats.last_position and stats.last_surface == surface.index then
        local dx = position.x - stats.last_position.x
        local dy = position.y - stats.last_position.y
        stats.distance = stats.distance + math.sqrt(dx * dx + dy * dy)
    end
    stats.last_position = {x = position.x, y = position.y}
    stats.last_surface = surface.index
end)

commands.add_command("fsm-players", "Returns a read-only player snapshot to RCON.", function(command)
    if command.player_index then
        local player = game.get_player(command.player_index)
        if not player or not player.admin then return end
    end
    rcon.print(response_prefix .. helpers.table_to_json(snapshot()))
end)

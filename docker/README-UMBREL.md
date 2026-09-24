# Umbrel deployment

The source checkout and runtime data deliberately live in separate directories:

```text
/home/umbrel/apps/factorio-server/source  # Git checkout
/home/umbrel/apps/factorio-server/data    # saves, mods, config and FSM database
```

Deploy or update after pushing changes:

```bash
cd /home/umbrel/apps/factorio-server/source
git pull --ff-only origin develop
docker compose -f compose.umbrel.yaml up -d --build
```

The panel listens on TCP port `3101` by default and Factorio listens on UDP port
`34197`. These can be overridden with `PANEL_PORT`, `PANEL_BIND_ADDRESS`, and
`FACTORIO_PORT`. Because data is bind-mounted from the sibling `data` directory,
rebuilding or replacing the container does not remove saves, mods, settings, or
users.


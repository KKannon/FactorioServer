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

Before the first OIDC-enabled deploy, create `.env` beside the compose file and
set the confidential client secret and the StupidMailCenter API key (never
commit this file):

```bash
cp .env.example .env
# Replace the placeholder secrets, then generate the internal mailer token:
printf '\nSTUPID_MAIL_INTERNAL_TOKEN=%s\n' "$(openssl rand -hex 32)" >> .env
```

Email delivery runs in the private `mailer` service; no mailer port is exposed
to the host or Cloudflare. It uses the vendored, pinned
`lib-stupidmailjavascript` backend client and five backend-owned templates:
first-access welcome, server lifecycle, worlds/backups, player access, and
system changes. A welcome email is recorded in the persistent SQLite database
and sent exactly once per OIDC user, with automatic retry after a delivery or
service failure. Successful
sensitive actions notify the authenticated operator at the email supplied by
OIDC; delivery failures are logged but never roll back the completed Factorio
action. The API key only needs the `email:send` scope.

The Monitoring page separates host, container, manager, and Factorio process
metrics. Samples are sent every five seconds through the authenticated
WebSocket; the REST endpoint remains a 30-second fallback. Player counts come
from Factorio's native join/leave log events. Live UPS is explicitly reported
as unavailable because Factorio does not expose it through a safe native
non-script command; script commands are not issued because they can affect
achievements.

To receive a single email when memory or the Factorio data volume crosses a
threshold, set `STUPID_MONITORING_ALERT_RECIPIENT`. Optional percentage
thresholds are `STUPID_MONITORING_MEMORY_THRESHOLD` and
`STUPID_MONITORING_DISK_THRESHOLD` (both default to `90`). An alert is sent only
on a threshold transition and is armed again after usage falls five percentage
points below the threshold.

The production callback is exactly
`https://factorio.stupidll.com/auth/callback`. The Cloudflare origin remains
`http://umbrel.local:3101`; TLS terminates at Cloudflare, while the browser sees
HTTPS and therefore accepts the Secure session cookie.

Authentication uses Authorization Code + PKCE S256. OIDC access, refresh, and
ID tokens are encrypted in the persistent SQLite database and are never sent to
the browser. Logout removes the local session before redirecting to the provider.
Users linked to the app can read status and configuration. Only roles listed in
`STUPID_AUTHENTICATOR_MANAGEMENT_ROLES` may change saves, mods, settings, or the
running server; the default is `admin,adm,manager,support,owner,operator`.
Only roles in `STUPID_AUTHENTICATOR_FACTORIO_ADMIN_ROLES` are automatically
promoted inside Factorio; the default is `admin,adm,owner,operator`.

The Players page edits Factorio's native whitelist, administrator list, and
banlist. When the game is running changes are also sent through RCON; if runtime
synchronization fails, the file change is rolled back. Whitelist enforcement is
enabled by default and its on/off policy is stored in the Factorio config volume.

The Worlds page imports and exports native Factorio `.zip` saves and stores
backups under `/opt/fsm-data/backups`. Creating, restoring, renaming, and deleting
worlds is blocked while Factorio is running. Delete and restore operations first
create a safety backup, so they never silently destroy the only existing copy.

The Map Generator reads `map-gen-settings.example.json` and
`map-settings.example.json` from the installed Factorio version. The simple
editor changes common fields, while the advanced editor exposes the complete
native documents. Factorio validates every request through `--create`; the
panel only reports success after the resulting save ZIP has been verified.
Map previews are also rendered by the installed Factorio binary and do not
create or modify a save.
Custom presets are stored under `/opt/fsm-data/map-presets`.

The Mods page parses native Factorio dependency declarations and reports
missing, disabled, version-mismatched, and conflicting dependencies before a
mod is enabled. An enabled mod cannot lose one of its required dependencies.
Modpacks can be created from the active installation, duplicated, exported,
loaded, and removed while the game server is stopped.

The panel listens on TCP port `3101` by default and Factorio listens on UDP port
`34197`. These can be overridden with `PANEL_PORT`, `PANEL_BIND_ADDRESS`, and
`FACTORIO_PORT`. Because data is bind-mounted from the sibling `data` directory,
rebuilding or replacing the container does not remove saves, mods, settings, or
users.


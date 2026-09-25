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
`lib-stupidmailjavascript` backend client and creates four versioned templates
on first startup: server lifecycle, worlds/backups, player access, and system
changes. Template IDs persist in the `mailer-data` volume. Successful sensitive
actions notify the authenticated operator at the email supplied by OIDC;
delivery failures are logged but never roll back the completed Factorio action.

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

The panel listens on TCP port `3101` by default and Factorio listens on UDP port
`34197`. These can be overridden with `PANEL_PORT`, `PANEL_BIND_ADDRESS`, and
`FACTORIO_PORT`. Because data is bind-mounted from the sibling `data` directory,
rebuilding or replacing the container does not remove saves, mods, settings, or
users.


# Security and administrative audit

The manager uses the StupidAuthenticator OIDC session as its identity source.
Every API route is authenticated. Configuration, RCON, player administration,
mods, logs, and server lifecycle routes additionally require a role listed in
`STUPID_AUTHENTICATOR_MANAGEMENT_ROLES`.

## Request protection

State-changing API calls require `X-FSM-Request: 1`, reject cross-site browser
requests, and are covered by a two MiB JSON body limit. Critical destructive
operations also require an `X-FSM-Confirm` value bound to the exact backend
route. The frontend adds that value only after the corresponding confirmation
dialog is accepted.

These checks supplement authentication and authorization; they do not replace
them. Save deletion continues to create a safety backup before removing the
world. Uploads retain their configured size limit and filename/path validation.

## Audit records

Administrative mutations and sensitive reads are stored in the existing SQLite
database. Records contain the actor's stable public identifier, display name,
role, Factorio username, action, normalized resource, result, HTTP status,
duration, and a request identifier. Request bodies, query strings, e-mail
addresses, cookies, passwords, access tokens, refresh tokens, and RCON secrets
are never stored in audit records.

Audit records are visible only to management roles under **Audit & security**.
They are retained for 180 days. The page also shows the effective permission and
protection summary without exposing configuration secrets.

Successful high-impact actions such as safe world deletion, removing mods, and
loading or deleting a modpack use the existing internal mailer notification
queue. Delivery remains best-effort and never changes the operation result.

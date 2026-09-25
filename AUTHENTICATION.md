# StupidAuthenticator integration

## Architecture and security decisions

- Authorization Code Flow uses PKCE S256, a cryptographically random `state`,
  `nonce`, and `code_verifier`. Login attempts expire after ten minutes and are
  deleted before the authorization code is exchanged, making `state` single-use.
- ID tokens are accepted only after RS256 signature/JWKS, issuer, audience,
  expiry, and nonce validation. UserInfo must have a stable `sub`,
  `public_user_id`, app link, and app role.
- The browser receives only an opaque `HttpOnly`, `SameSite=Lax` session cookie.
  It is `Secure` for the production HTTPS callback. Access, refresh, and ID
  tokens are AES-GCM encrypted in SQLite using the existing server key.
- The confidential client secret and tokens are never logged or returned to the
  browser. The client secret exists only in the untracked `.env` on the server.
- UserInfo is the identity source. The local session caches the normalized
  `AuthUser` only as a fallback and replaces it after token refresh or an
  explicit foreground refresh. A UserInfo 401 causes exactly one token refresh;
  another failure deletes the local session.
- Read access requires a valid app role. Mutations additionally require a role
  from `STUPID_AUTHENTICATOR_MANAGEMENT_ROLES` and are enforced in Go middleware.
- Panel management and in-game administrator privileges are separate. Roles in
  `STUPID_AUTHENTICATOR_FACTORIO_ADMIN_ROLES` are promoted through Factorio's
  native admin list; support and manager roles can manage the panel without
  automatically becoming administrators inside the game.
- Logout deletes the local database session and cookie before redirecting to the
  provider. No unregistered `post_logout_redirect_uri` is sent.

## Required environment

Copy `.env.example` to `.env`. Set the secret supplied separately:

```dotenv
STUPID_AUTHENTICATOR_CLIENT_SECRET=CLIENT_SECRET_PROVIDED_SEPARATELY
STUPID_AUTHENTICATOR_ISSUER=https://authenticator.stupidll.com/o
STUPID_AUTHENTICATOR_REDIRECT_URI=https://factorio.stupidll.com/auth/callback
STUPID_AUTHENTICATOR_MANAGEMENT_ROLES=admin,adm,manager,support,owner,operator
STUPID_AUTHENTICATOR_FACTORIO_ADMIN_ROLES=admin,adm,owner,operator
```

The issuer includes `/o` because it must exactly match the provider discovery
document and the `iss` claim in ID tokens. The browser-facing provider base URL
remains `https://authenticator.stupidll.com`.

The startup validation accepts only the three registered callback URLs from the
provider configuration. Production should use the HTTPS callback above.

## Tests

Frontend preferences, avatar behavior, and production bundle:

```bash
npm ci
npm test
npm run build
npm audit --omit=dev
```

Focused Go tests (Linux/macOS):

```bash
test_root="$(mktemp -d)"
mkdir -p "$test_root"/{config,mods,saves,mod_packs}
cp conf.json.example "$test_root/conf.json"
cd src
FSM_DIR="$test_root" \
FSM_CONF="$test_root/conf.json" \
FSM_MODPACK_DIR="$test_root/mod_packs" \
go test ./api -run 'Test(Authorization|OIDC|Preferences|Safe|UserInfo|Configuration|Management)' -count=1
```

The older full mod-portal test suite additionally requires valid Factorio portal
test credentials; without them its download tests fail independently of OIDC.

## Production verification

1. Put the secret in `/home/umbrel/apps/factorio-server/source/.env`.
2. Run the documented `git pull` and `docker compose ... up -d --build` command.
3. Open `https://factorio.stupidll.com` and select **Entrar com
   StupidAuthenticator**.
4. Confirm the provider returns to the exact `/auth/callback` URL and the panel
   shows the provider name, role, and versioned avatar (or initials).
5. Change a preference in StupidAuthenticator, leave the tab in the background
   for at least 15 minutes, and return. Confirm theme/language/avatar update.
6. Log out and confirm the local session is rejected before the browser reaches
   the provider logout page.

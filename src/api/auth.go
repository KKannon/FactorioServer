package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/glebarez/sqlite"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	defaultProviderBase  = "https://authenticator.stupidll.com"
	defaultIssuer        = defaultProviderBase + "/o"
	defaultClientID      = "laXVUHiwiUGnNVyMrGKqYf03V1mdIS3JlU5Vk4ka"
	defaultAppSlug       = "factorio-server-manager"
	loginAttemptLifetime = 10 * time.Minute
	sessionLifetime      = 12 * time.Hour
)

type OIDCConfig struct {
	Issuer, ClientID, ClientSecret, RedirectURI, AppSlug        string
	AuthorizationURL, TokenURL, UserInfoURL, JWKSURL, LogoutURL string
}

type LoginAttempt struct {
	StateHash    string `gorm:"primaryKey;size:64"`
	Nonce        string `gorm:"not null"`
	CodeVerifier string `gorm:"not null"`
	ReturnPath   string
	ExpiresAt    time.Time `gorm:"index"`
}

type OIDCSession struct {
	IDHash                                       string `gorm:"primaryKey;size:64"`
	UserJSON, AccessToken, RefreshToken, IDToken []byte
	TokenExpiry, ProfileSyncedAt                 time.Time
	ExpiresAt                                    time.Time `gorm:"index"`
}

type authContextKey struct{}

type Auth struct {
	config             OIDCConfig
	db                 *gorm.DB
	oauth              oauth2.Config
	verifier           *oidc.IDTokenVerifier
	httpClient         *http.Client
	cipher             cipher.AEAD
	cookieName         string
	cookieSecure       bool
	managementRoles    map[string]struct{}
	factorioAdminRoles map[string]struct{}
}

var auth Auth

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadOIDCConfig() (OIDCConfig, error) {
	issuer := strings.TrimRight(envOrDefault("STUPID_AUTHENTICATOR_ISSUER", defaultIssuer), "/")
	config := OIDCConfig{
		Issuer: issuer, ClientID: envOrDefault("STUPID_AUTHENTICATOR_CLIENT_ID", defaultClientID),
		ClientSecret: os.Getenv("STUPID_AUTHENTICATOR_CLIENT_SECRET"), RedirectURI: os.Getenv("STUPID_AUTHENTICATOR_REDIRECT_URI"),
		AppSlug:          envOrDefault("STUPID_AUTHENTICATOR_APP_SLUG", defaultAppSlug),
		AuthorizationURL: envOrDefault("STUPID_AUTHENTICATOR_AUTHORIZATION_URL", defaultProviderBase+"/o/authorize/"),
		TokenURL:         envOrDefault("STUPID_AUTHENTICATOR_TOKEN_URL", defaultProviderBase+"/o/token/"),
		UserInfoURL:      envOrDefault("STUPID_AUTHENTICATOR_USERINFO_URL", defaultProviderBase+"/o/userinfo/"),
		JWKSURL:          envOrDefault("STUPID_AUTHENTICATOR_JWKS_URL", defaultProviderBase+"/o/.well-known/jwks.json"),
		LogoutURL:        envOrDefault("STUPID_AUTHENTICATOR_LOGOUT_URL", defaultProviderBase+"/o/logout/"),
	}
	if config.ClientSecret == "" {
		return config, errors.New("STUPID_AUTHENTICATOR_CLIENT_SECRET is required")
	}
	authorizedRedirects := map[string]bool{
		"https://factorio.stupidll.com/auth/callback": true,
		"http://factorio.stupidll.com/auth/callback":  true,
		"http://192.168.1.14:3101/auth/callback":      true,
	}
	if !authorizedRedirects[config.RedirectURI] {
		return config, errors.New("STUPID_AUTHENTICATOR_REDIRECT_URI is not in the authorized redirect URI list")
	}
	return config, nil
}

func SetupAuth() {
	config, err := loadOIDCConfig()
	if err != nil {
		log.Fatalf("OIDC configuration error: %v", err)
	}
	appConfig := bootstrap.GetConfig()
	key, err := base64.StdEncoding.DecodeString(appConfig.CookieEncryptionKey)
	if err != nil || len(key) < 32 {
		log.Fatal("FSM cookie encryption key is invalid")
	}
	db, err := gorm.Open(sqlite.Open(appConfig.SQLiteDatabaseFile), nil)
	if err != nil {
		log.Fatalf("open authentication database: %v", err)
	}
	if err := db.AutoMigrate(&LoginAttempt{}, &OIDCSession{}, &WelcomeNotification{}, &AuditEvent{}); err != nil {
		log.Fatalf("migrate authentication database: %v", err)
	}
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		log.Fatalf("initialize session encryption: %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("initialize session encryption: %v", err)
	}
	redirect, _ := url.Parse(config.RedirectURI)
	auth = newAuth(config, db, aead, redirect.Scheme == "https")
}

func newAuth(config OIDCConfig, db *gorm.DB, aead cipher.AEAD, secure bool) Auth {
	oauthConfig := oauth2.Config{ClientID: config.ClientID, ClientSecret: config.ClientSecret, RedirectURL: config.RedirectURI,
		Scopes:   []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint: oauth2.Endpoint{AuthURL: config.AuthorizationURL, TokenURL: config.TokenURL, AuthStyle: oauth2.AuthStyleInParams}}
	cookieName := "fsm_session"
	if secure {
		cookieName = "__Host-fsm_session"
	}
	managementRoles := make(map[string]struct{})
	for _, role := range strings.Split(envOrDefault("STUPID_AUTHENTICATOR_MANAGEMENT_ROLES", "admin,adm,manager,support,owner,operator"), ",") {
		if role = strings.TrimSpace(role); role != "" {
			managementRoles[role] = struct{}{}
		}
	}
	factorioAdminRoles := make(map[string]struct{})
	for _, role := range strings.Split(envOrDefault("STUPID_AUTHENTICATOR_FACTORIO_ADMIN_ROLES", "admin,adm,owner,operator"), ",") {
		if role = strings.TrimSpace(role); role != "" {
			factorioAdminRoles[role] = struct{}{}
		}
	}
	return Auth{config: config, db: db, oauth: oauthConfig,
		verifier:   oidc.NewVerifier(config.Issuer, oidc.NewRemoteKeySet(context.Background(), config.JWKSURL), &oidc.Config{ClientID: config.ClientID}),
		httpClient: &http.Client{Timeout: 15 * time.Second}, cipher: aead, cookieName: cookieName, cookieSecure: secure, managementRoles: managementRoles, factorioAdminRoles: factorioAdminRoles}
}

func (a *Auth) prepareUser(user AuthUser) AuthUser {
	_, user.CanManage = a.managementRoles[user.Role]
	_, user.ServerAdmin = a.factorioAdminRoles[user.Role]
	user.GameUsername = user.FactorioUsername()
	// Keep the public response consistent even with providers that only expose
	// the standard OIDC preferred_username alias.
	if strings.TrimSpace(user.Username) == "" {
		user.Username = strings.TrimSpace(user.PreferredUsername)
	}
	if user.GameUsername != "" {
		if err := factorio.EnsurePlayerAccess(user.GameUsername, user.ServerAdmin); err != nil {
			log.Printf("could not synchronize Factorio access for user %s: %v", user.PublicUserID, err)
		}
	}
	return user
}

func randomURLSafe(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func safeReturnPath(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return value
}

func (a *Auth) authorizationURL(returnPath string) (string, error) {
	a.db.Where("expires_at < ?", time.Now()).Delete(&LoginAttempt{})
	state, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomURLSafe(64)
	if err != nil {
		return "", err
	}
	attempt := LoginAttempt{StateHash: hashValue(state), Nonce: nonce, CodeVerifier: verifier, ReturnPath: safeReturnPath(returnPath), ExpiresAt: time.Now().Add(loginAttemptLifetime)}
	if err := a.db.Create(&attempt).Error; err != nil {
		return "", err
	}
	return a.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)), oauth2.SetAuthURLParam("code_challenge_method", "S256")), nil
}

func (a *Auth) consumeAttempt(state string) (LoginAttempt, error) {
	if state == "" {
		return LoginAttempt{}, errors.New("missing state")
	}
	var attempt LoginAttempt
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("state_hash = ?", hashValue(state)).First(&attempt).Error; err != nil {
			return err
		}
		return tx.Delete(&attempt).Error
	})
	if err != nil || time.Now().After(attempt.ExpiresAt) {
		return LoginAttempt{}, errors.New("invalid or expired state")
	}
	return attempt, nil
}

func (a *Auth) seal(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	nonce := make([]byte, a.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return a.cipher.Seal(nonce, nonce, []byte(value), nil), nil
}

func (a *Auth) open(value []byte) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	nonceSize := a.cipher.NonceSize()
	if len(value) < nonceSize {
		return "", errors.New("invalid encrypted session value")
	}
	plain, err := a.cipher.Open(nil, value[:nonceSize], value[nonceSize:], nil)
	return string(plain), err
}

func (a *Auth) fetchUserInfo(ctx context.Context, accessToken string) (AuthUser, int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.config.UserInfoURL, nil)
	if err != nil {
		return AuthUser{}, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := a.httpClient.Do(request)
	if err != nil {
		return AuthUser{}, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return AuthUser{}, response.StatusCode, fmt.Errorf("userinfo returned status %d", response.StatusCode)
	}
	var user AuthUser
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&user); err != nil {
		return AuthUser{}, response.StatusCode, err
	}
	if err := user.Validate(a.config.AppSlug); err != nil {
		return AuthUser{}, response.StatusCode, err
	}
	return user, response.StatusCode, nil
}

func (a *Auth) saveSession(w http.ResponseWriter, token *oauth2.Token, rawIDToken string, user AuthUser) error {
	rawSession, err := randomURLSafe(32)
	if err != nil {
		return err
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	access, err := a.seal(token.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := a.seal(token.RefreshToken)
	if err != nil {
		return err
	}
	idToken, err := a.seal(rawIDToken)
	if err != nil {
		return err
	}
	session := OIDCSession{IDHash: hashValue(rawSession), UserJSON: userJSON, AccessToken: access, RefreshToken: refresh, IDToken: idToken, TokenExpiry: token.Expiry, ProfileSyncedAt: time.Now(), ExpiresAt: time.Now().Add(sessionLifetime)}
	if err := a.db.Create(&session).Error; err != nil {
		return err
	}
	a.setSessionCookie(w, rawSession)
	return nil
}

func (a *Auth) setSessionCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{Name: a.cookieName, Value: value, Path: "/", HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionLifetime.Seconds())})
}

func (a *Auth) clearSession(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		return "", nil
	}
	var session OIDCSession
	if err := a.db.Where("id_hash = ?", hashValue(cookie.Value)).First(&session).Error; err == nil {
		a.db.Delete(&session)
	}
	http.SetCookie(w, &http.Cookie{Name: a.cookieName, Value: "", Path: "/", HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	return a.open(session.IDToken)
}

func (a *Auth) sessionFromRequest(r *http.Request) (*OIDCSession, AuthUser, error) {
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		return nil, AuthUser{}, err
	}
	var session OIDCSession
	if err := a.db.Where("id_hash = ?", hashValue(cookie.Value)).First(&session).Error; err != nil {
		return nil, AuthUser{}, err
	}
	if time.Now().After(session.ExpiresAt) {
		a.db.Delete(&session)
		return nil, AuthUser{}, errors.New("session expired")
	}
	var user AuthUser
	if err := json.Unmarshal(session.UserJSON, &user); err != nil {
		return nil, AuthUser{}, err
	}
	return &session, user, nil
}

func (a *Auth) refreshProfile(ctx context.Context, session *OIDCSession, force bool) (AuthUser, error) {
	access, err := a.open(session.AccessToken)
	if err != nil {
		return AuthUser{}, err
	}
	refresh, err := a.open(session.RefreshToken)
	if err != nil {
		return AuthUser{}, err
	}
	needsRefresh := access == "" || (!session.TokenExpiry.IsZero() && time.Now().Add(time.Minute).After(session.TokenExpiry))
	token := &oauth2.Token{AccessToken: access, RefreshToken: refresh, Expiry: session.TokenExpiry}
	if needsRefresh {
		if refresh == "" {
			return AuthUser{}, errors.New("session token expired")
		}
		token, err = a.oauth.TokenSource(ctx, token).Token()
		if err != nil {
			return AuthUser{}, errors.New("refresh token rejected")
		}
	}
	user, status, infoErr := a.fetchUserInfo(ctx, token.AccessToken)
	if infoErr != nil && status == http.StatusUnauthorized && !needsRefresh && refresh != "" {
		token.Expiry = time.Unix(1, 0)
		token, err = a.oauth.TokenSource(ctx, token).Token()
		if err == nil {
			user, _, infoErr = a.fetchUserInfo(ctx, token.AccessToken)
		}
	}
	if infoErr != nil {
		return AuthUser{}, infoErr
	}
	userJSON, _ := json.Marshal(user)
	session.UserJSON = userJSON
	session.ProfileSyncedAt = time.Now()
	session.TokenExpiry = token.Expiry
	session.ExpiresAt = time.Now().Add(sessionLifetime)
	session.AccessToken, _ = a.seal(token.AccessToken)
	session.RefreshToken, _ = a.seal(token.RefreshToken)
	if err := a.db.Save(session).Error; err != nil {
		return AuthUser{}, err
	}
	return user, nil
}

func BeginOIDCLogin(w http.ResponseWriter, r *http.Request) {
	location, err := auth.authorizationURL(r.URL.Query().Get("return"))
	if err != nil {
		http.Error(w, "Unable to start sign in", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, location, http.StatusFound)
}

func OIDCCallback(w http.ResponseWriter, r *http.Request) {
	attempt, err := auth.consumeAttempt(r.URL.Query().Get("state"))
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid_state", http.StatusSeeOther)
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		if providerError == "access_denied" {
			http.Redirect(w, r, "/login?error=access_denied", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/login?error=provider", http.StatusSeeOther)
		}
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?error=missing_code", http.StatusSeeOther)
		return
	}
	token, err := auth.oauth.Exchange(r.Context(), code, oauth2.SetAuthURLParam("code_verifier", attempt.CodeVerifier))
	if err != nil {
		// Keep enough information to diagnose provider/configuration failures without
		// ever logging the authorization code, PKCE verifier, client secret or tokens.
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			log.Printf("OIDC token exchange failed: status=%d oauth_error=%q verifier_length=%d", retrieveErr.Response.StatusCode, retrieveErr.ErrorCode, len(attempt.CodeVerifier))
		} else {
			log.Printf("OIDC token exchange request failed: error_type=%T verifier_length=%d", err, len(attempt.CodeVerifier))
		}
		http.Redirect(w, r, "/login?error=token_exchange", http.StatusSeeOther)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusSeeOther)
		return
	}
	idToken, err := auth.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusSeeOther)
		return
	}
	var claims struct {
		Nonce string `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(attempt.Nonce)) != 1 {
		http.Redirect(w, r, "/login?error=invalid_nonce", http.StatusSeeOther)
		return
	}
	user, _, err := auth.fetchUserInfo(r.Context(), token.AccessToken)
	if err != nil || user.Subject != idToken.Subject {
		http.Redirect(w, r, "/login?error=userinfo", http.StatusSeeOther)
		return
	}
	user = auth.prepareUser(user)
	if err := auth.saveSession(w, token, rawIDToken, user); err != nil {
		http.Error(w, "Unable to create session", http.StatusInternalServerError)
		return
	}
	enqueueWelcomeNotification(user)
	http.Redirect(w, r, attempt.ReturnPath, http.StatusSeeOther)
}

func OIDCLogout(w http.ResponseWriter, r *http.Request) {
	idToken, _ := auth.clearSession(w, r)
	logout, _ := url.Parse(auth.config.LogoutURL)
	query := logout.Query()
	query.Set("client_id", auth.config.ClientID)
	if idToken != "" {
		query.Set("id_token_hint", idToken)
	}
	logout.RawQuery = query.Encode()
	http.Redirect(w, r, logout.String(), http.StatusFound)
}

func CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(authContextKey{}).(AuthUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Always source username and other mutable profile fields from UserInfo on
	// page load. This also repairs sessions created before username was mapped.
	if session, _, err := auth.sessionFromRequest(r); err == nil {
		if refreshed, refreshErr := auth.refreshProfile(r.Context(), session, true); refreshErr == nil {
			user = refreshed
		} else {
			log.Printf("could not refresh current user profile: %v", refreshErr)
		}
	}
	user = auth.prepareUser(user)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(user)
}

func RefreshCurrentUser(w http.ResponseWriter, r *http.Request) {
	session, _, err := auth.sessionFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := auth.refreshProfile(r.Context(), session, true)
	if err != nil {
		auth.clearSession(w, r)
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}
	user = auth.prepareUser(user)
	if cookie, cookieErr := r.Cookie(auth.cookieName); cookieErr == nil {
		auth.setSessionCookie(w, cookie.Value)
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(user)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, user, err := auth.sessionFromRequest(r)
		if err != nil {
			if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/ws" {
				http.Redirect(w, r, "/login?return="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
			} else {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			}
			return
		}
		if time.Now().Add(time.Minute).After(session.TokenExpiry) {
			user, err = auth.refreshProfile(r.Context(), session, false)
			if err != nil {
				auth.clearSession(w, r)
				http.Error(w, "Session expired", http.StatusUnauthorized)
				return
			}
			if cookie, cookieErr := r.Cookie(auth.cookieName); cookieErr == nil {
				auth.setSessionCookie(w, cookie.Value)
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey{}, user)))
	})
}

func RequireManagementRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-FSM-Request") != "1" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		user, ok := r.Context().Value(authContextKey{}).(AuthUser)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if _, allowed := auth.managementRoles[user.Role]; !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

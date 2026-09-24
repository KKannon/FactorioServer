package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testCipher(t *testing.T) cipher.AEAD {
	block, err := aes.NewCipher(make([]byte, 32))
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	return aead
}

func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())), nil)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&LoginAttempt{}, &OIDCSession{}))
	return db
}

func TestAuthorizationURLUsesPKCEStateNonceAndExactRedirect(t *testing.T) {
	config := OIDCConfig{Issuer: "https://issuer.example", ClientID: "client", ClientSecret: "secret", RedirectURI: "https://factorio.stupidll.com/auth/callback", AppSlug: defaultAppSlug, AuthorizationURL: "https://issuer.example/o/authorize/", TokenURL: "https://issuer.example/o/token/", JWKSURL: "https://issuer.example/jwks", UserInfoURL: "https://issuer.example/userinfo"}
	a := newAuth(config, testDB(t), testCipher(t), true)
	location, err := a.authorizationURL("/mods")
	require.NoError(t, err)
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	query := parsed.Query()
	require.Equal(t, "code", query.Get("response_type"))
	require.Equal(t, config.ClientID, query.Get("client_id"))
	require.Equal(t, config.RedirectURI, query.Get("redirect_uri"))
	require.Equal(t, "openid profile email", query.Get("scope"))
	require.Equal(t, "S256", query.Get("code_challenge_method"))
	require.NotEmpty(t, query.Get("code_challenge"))
	require.NotEmpty(t, query.Get("nonce"))
	attempt, err := a.consumeAttempt(query.Get("state"))
	require.NoError(t, err)
	require.Equal(t, "/mods", attempt.ReturnPath)
	_, err = a.consumeAttempt(query.Get("state"))
	require.Error(t, err, "state must be single-use")
}

func TestOIDCCallbackCreatesServerSessionAndLogoutDeletesIt(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicJWK := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "test-key", Algorithm: string(jose.RS256), Use: "sig"}
	var signedIDToken, expectedVerifier string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []jose.JSONWebKey{publicJWK}})
		case "/token":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "secret", r.Form.Get("client_secret"))
			require.Equal(t, expectedVerifier, r.Form.Get("code_verifier"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "access-secret", "refresh_token": "refresh-secret", "token_type": "Bearer", "expires_in": 3600, "id_token": signedIDToken})
		case "/userinfo":
			require.Equal(t, "Bearer access-secret", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"sub": "subject-1", "public_user_id": "11111111-1111-1111-1111-111111111111", "email": "user@example.com", "name": "Test User", "role": "admin", "roles": map[string]string{defaultAppSlug: "admin"}, "preferences": map[string]interface{}{"theme": "system", "accent_color": "#E39827", "language": "pt-BR", "timezone": "America/Sao_Paulo", "date_format": "DD/MM/YYYY"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	config := OIDCConfig{Issuer: provider.URL, ClientID: "client", ClientSecret: "secret", RedirectURI: "https://factorio.stupidll.com/auth/callback", AppSlug: defaultAppSlug, AuthorizationURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token", UserInfoURL: provider.URL + "/userinfo", JWKSURL: provider.URL + "/jwks", LogoutURL: provider.URL + "/logout"}
	auth = newAuth(config, testDB(t), testCipher(t), true)
	location, err := auth.authorizationURL("/saves")
	require.NoError(t, err)
	parsed, _ := url.Parse(location)
	state := parsed.Query().Get("state")
	var attempt LoginAttempt
	require.NoError(t, auth.db.Where("state_hash = ?", hashValue(state)).First(&attempt).Error)
	expectedVerifier = attempt.CodeVerifier
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"))
	require.NoError(t, err)
	signedIDToken, err = jwt.Signed(signer).Claims(jwt.Claims{Issuer: provider.URL, Subject: "subject-1", Audience: jwt.Audience{"client"}, Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}).Claims(struct {
		Nonce string `json:"nonce"`
	}{Nonce: attempt.Nonce}).Serialize()
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/callback?state="+url.QueryEscape(state)+"&code=valid", nil)
	OIDCCallback(recorder, request)
	require.Equal(t, http.StatusSeeOther, recorder.Code)
	require.Equal(t, "/saves", recorder.Header().Get("Location"))
	require.Len(t, recorder.Result().Cookies(), 1)
	cookie := recorder.Result().Cookies()[0]
	require.True(t, cookie.HttpOnly)
	require.True(t, cookie.Secure)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	var session OIDCSession
	require.NoError(t, auth.db.Where("id_hash = ?", hashValue(cookie.Value)).First(&session).Error)
	require.NotContains(t, string(session.AccessToken), "access-secret")
	statusRecorder := httptest.NewRecorder()
	statusRequest := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	statusRequest.AddCookie(cookie)
	AuthMiddleware(http.HandlerFunc(CurrentUser)).ServeHTTP(statusRecorder, statusRequest)
	require.Equal(t, http.StatusOK, statusRecorder.Code)
	require.Contains(t, statusRecorder.Body.String(), "public_user_id")
	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	logoutRequest.AddCookie(cookie)
	OIDCLogout(logoutRecorder, logoutRequest)
	require.Equal(t, http.StatusFound, logoutRecorder.Code)
	require.True(t, strings.HasPrefix(logoutRecorder.Header().Get("Location"), provider.URL+"/logout?"))
	require.Error(t, auth.db.Where("id_hash = ?", hashValue(cookie.Value)).First(&OIDCSession{}).Error)
}

func TestPreferencesMappingPreservesUnknownFieldsAndNormalizes(t *testing.T) {
	raw := `{"sub":"s","public_user_id":"p","role":"admin","preferences":{"theme":"system","accent_color":"not-a-color","language":"es-ES","timezone":"Europe/Madrid","date_format":"YYYY-MM-DD","future_option":{"enabled":true}}}`
	var user AuthUser
	require.NoError(t, json.Unmarshal([]byte(raw), &user))
	require.Equal(t, "system", user.Preferences.Theme)
	require.Equal(t, "#E39827", user.Preferences.AccentColor)
	require.Equal(t, "es-ES", user.Preferences.Language)
	require.Equal(t, "Europe/Madrid", user.Preferences.Timezone)
	encoded, err := json.Marshal(user)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "future_option")
	require.Equal(t, "TU", (AuthUser{Name: "Test User"}).Initials())
	require.Equal(t, "?", (AuthUser{}).Initials())
}

func TestUserInfo401RefreshesOnceAndUpdatesProfile(t *testing.T) {
	userinfoCalls, tokenCalls := 0, 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/userinfo":
			userinfoCalls++
			if r.Header.Get("Authorization") != "Bearer new-access" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"sub": "subject-1", "public_user_id": "public-1", "name": "Updated User", "picture": "https://cdn.example/avatar.png?v=3", "role": "admin", "roles": map[string]string{defaultAppSlug: "admin"}, "preferences": map[string]interface{}{"theme": "dark"}})
		case "/token":
			tokenCalls++
			require.NoError(t, r.ParseForm())
			require.Equal(t, "refresh-secret", r.Form.Get("refresh_token"))
			require.Equal(t, "secret", r.Form.Get("client_secret"))
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "new-access", "refresh_token": "new-refresh", "token_type": "Bearer", "expires_in": 3600})
		}
	}))
	defer provider.Close()
	config := OIDCConfig{Issuer: provider.URL, ClientID: "client", ClientSecret: "secret", RedirectURI: "https://factorio.stupidll.com/auth/callback", AppSlug: defaultAppSlug, TokenURL: provider.URL + "/token", UserInfoURL: provider.URL + "/userinfo", JWKSURL: provider.URL + "/jwks"}
	a := newAuth(config, testDB(t), testCipher(t), true)
	oldAccess, _ := a.seal("old-access")
	refresh, _ := a.seal("refresh-secret")
	session := OIDCSession{IDHash: "session", AccessToken: oldAccess, RefreshToken: refresh, TokenExpiry: time.Now().Add(time.Hour), ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, a.db.Create(&session).Error)
	user, err := a.refreshProfile(context.Background(), &session, true)
	require.NoError(t, err)
	require.Equal(t, 2, userinfoCalls)
	require.Equal(t, 1, tokenCalls)
	require.Equal(t, "Updated User", user.Name)
	require.Contains(t, user.Picture, "v=3")
	storedAccess, err := a.open(session.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "new-access", storedAccess)
}

func TestSafeReturnPathRejectsExternalURLs(t *testing.T) {
	require.Equal(t, "/", safeReturnPath("https://evil.example"))
	require.Equal(t, "/", safeReturnPath("//evil.example"))
	require.Equal(t, "/", safeReturnPath(`/\evil.example`))
	require.Equal(t, "/mods?tab=all", safeReturnPath("/mods?tab=all"))
}

func TestConfigurationAcceptsOnlyRegisteredRedirects(t *testing.T) {
	t.Setenv("STUPID_AUTHENTICATOR_CLIENT_SECRET", "secret")
	t.Setenv("STUPID_AUTHENTICATOR_ISSUER", "")
	t.Setenv("STUPID_AUTHENTICATOR_REDIRECT_URI", "https://evil.example/auth/callback")
	_, err := loadOIDCConfig()
	require.Error(t, err)
	t.Setenv("STUPID_AUTHENTICATOR_REDIRECT_URI", "https://factorio.stupidll.com/auth/callback")
	config, err := loadOIDCConfig()
	require.NoError(t, err)
	require.Equal(t, defaultClientID, config.ClientID)
	require.Equal(t, "https://authenticator.stupidll.com/o", config.Issuer)
}

func TestManagementAuthorizationIsEnforcedServerSide(t *testing.T) {
	auth.managementRoles = map[string]struct{}{"admin": {}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	viewerRequest := httptest.NewRequest(http.MethodPost, "/api/server/start", nil)
	viewerRequest.Header.Set("X-FSM-Request", "1")
	viewerRequest = viewerRequest.WithContext(context.WithValue(viewerRequest.Context(), authContextKey{}, AuthUser{Role: "viewer"}))
	viewerResponse := httptest.NewRecorder()
	RequireManagementRole(next).ServeHTTP(viewerResponse, viewerRequest)
	require.Equal(t, http.StatusForbidden, viewerResponse.Code)
	adminRequest := httptest.NewRequest(http.MethodPost, "/api/server/start", nil)
	adminRequest.Header.Set("X-FSM-Request", "1")
	adminRequest = adminRequest.WithContext(context.WithValue(adminRequest.Context(), authContextKey{}, AuthUser{Role: "admin"}))
	adminResponse := httptest.NewRecorder()
	RequireManagementRole(next).ServeHTTP(adminResponse, adminRequest)
	require.Equal(t, http.StatusNoContent, adminResponse.Code)
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func TestSecurityHeadersDisableCaching(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/bundle.js", nil)
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store")
	}
	csp := recorder.Header().Get("Content-Security-Policy")
	for _, required := range []string{"img-src 'self' https: data: blob:", "https://static.cloudflareinsights.com", "https://cloudflareinsights.com"} {
		if !strings.Contains(csp, required) {
			t.Fatalf("Content-Security-Policy %q does not contain %q", csp, required)
		}
	}
	if recorder.Header().Get("Strict-Transport-Security") == "" || recorder.Header().Get("Permissions-Policy") == "" {
		t.Fatal("security hardening headers are missing")
	}
}

func TestPanelRequestMiddlewareProtectsStateChanges(t *testing.T) {
	handler := PanelRequestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	missing := httptest.NewRequest(http.MethodPost, "https://factorio.stupidll.com/api/server/start", nil)
	missing.Host = "factorio.stupidll.com"
	missingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(missingRecorder, missing)
	if missingRecorder.Code != http.StatusForbidden {
		t.Fatalf("missing panel header status = %d", missingRecorder.Code)
	}

	valid := httptest.NewRequest(http.MethodPost, "https://factorio.stupidll.com/api/server/start", nil)
	valid.Host = "factorio.stupidll.com"
	valid.Header.Set("X-FSM-Request", "1")
	valid.Header.Set("Origin", "https://factorio.stupidll.com")
	validRecorder := httptest.NewRecorder()
	handler.ServeHTTP(validRecorder, valid)
	if validRecorder.Code != http.StatusNoContent {
		t.Fatalf("same-origin request status = %d", validRecorder.Code)
	}

	crossSite := httptest.NewRequest(http.MethodPost, "https://factorio.stupidll.com/api/server/start", nil)
	crossSite.Host = "factorio.stupidll.com"
	crossSite.Header.Set("X-FSM-Request", "1")
	crossSite.Header.Set("Origin", "https://evil.example")
	crossSiteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(crossSiteRecorder, crossSite)
	if crossSiteRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-site request status = %d", crossSiteRecorder.Code)
	}
}

func TestDestructiveConfirmationMiddleware(t *testing.T) {
	handler := RequireDestructiveConfirmation("RemoveSave", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := httptest.NewRequest(http.MethodDelete, "/api/saves/rm/world.zip", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing confirmation status = %d", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/saves/rm/world.zip", nil)
	request.Header.Set("X-FSM-Confirm", "RemoveSave")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("confirmed request status = %d", recorder.Code)
	}
}

func TestAuditMiddlewarePersistsOnlySafeMetadata(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.AutoMigrate(&AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	previous := auth
	auth.db = database
	defer func() { auth = previous }()

	request := httptest.NewRequest(http.MethodDelete, "/api/saves/rm/world.zip?token=must-not-be-stored", strings.NewReader(`{"password":"must-not-be-stored"}`))
	request = mux.SetURLVars(request, map[string]string{"save": "world.zip"})
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, AuthUser{PublicUserID: "public-1", Name: "Admin", Role: "owner", Username: "Engineer"}))
	recorder := httptest.NewRecorder()
	AuditMiddleware("RemoveSave", http.MethodDelete, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(recorder, request)

	var event AuditEvent
	if err = database.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Action != "RemoveSave" || event.Resource != "world.zip" || event.ActorID != "public-1" || event.Result != "success" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	serialized := event.Action + event.Resource + event.ActorName + event.Error
	if strings.Contains(serialized, "must-not-be-stored") || strings.Contains(serialized, "password") || strings.Contains(serialized, "token") {
		t.Fatalf("audit event persisted sensitive request data: %#v", event)
	}
}

func TestReadRequestBodyRejectsOversizedJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/settings/update", strings.NewReader(strings.Repeat("x", maxJSONBodySize+1)))
	recorder := httptest.NewRecorder()
	_, _, err := ReadRequestBody(recorder, request)
	if err == nil {
		t.Fatal("oversized body was accepted")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d", recorder.Code)
	}
}

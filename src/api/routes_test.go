package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
}

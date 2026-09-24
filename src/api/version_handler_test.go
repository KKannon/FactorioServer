package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstallFactorioVersionRejectsInvalidInput(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/server/version", strings.NewReader(`{"version":"../../tmp"}`))
	recorder := httptest.NewRecorder()
	InstallFactorioVersion(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestInstallFactorioVersionRejectsMalformedJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/server/version", strings.NewReader(`{"version":`))
	recorder := httptest.NewRecorder()
	InstallFactorioVersion(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

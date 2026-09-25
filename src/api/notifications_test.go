package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestNotificationMiddlewareQueuesSuccessfulManagementEvent(t *testing.T) {
	previous := notifications
	notifications = &notificationDispatcher{queue: make(chan notificationPayload, 1)}
	t.Cleanup(func() { notifications = previous })
	request := httptest.NewRequest(http.MethodPost, "/api/saves/world.zip/backup", nil)
	request = mux.SetURLVars(request, map[string]string{"save": "world.zip"})
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, AuthUser{Email: "admin@example.com", Name: "Admin", Role: "owner"}))
	recorder := httptest.NewRecorder()
	NotificationMiddleware("CreateSaveBackup", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})).ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
	select {
	case payload := <-notifications.queue:
		require.Equal(t, "CreateSaveBackup", payload.Event)
		require.Equal(t, "admin@example.com", payload.Recipient)
		require.Equal(t, "world.zip", payload.Resource)
	default:
		t.Fatal("expected notification to be queued")
	}
}

func TestNotificationMiddlewareDoesNotQueueFailedEvent(t *testing.T) {
	previous := notifications
	notifications = &notificationDispatcher{queue: make(chan notificationPayload, 1)}
	t.Cleanup(func() { notifications = previous })
	request := httptest.NewRequest(http.MethodPost, "/api/server/start", nil)
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, AuthUser{Email: "admin@example.com", Name: "Admin", Role: "owner"}))
	recorder := httptest.NewRecorder()
	NotificationMiddleware("StartServer", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed", http.StatusConflict)
	})).ServeHTTP(recorder, request)
	require.Len(t, notifications.queue, 0)
}

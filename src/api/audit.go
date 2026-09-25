package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const auditRetention = 180 * 24 * time.Hour

// AuditEvent deliberately stores no request body, query string, cookie, token,
// password, or e-mail address. Route metadata and the normalized resource are
// sufficient to reconstruct administrative activity without persisting secrets.
type AuditEvent struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
	RequestID      string    `json:"request_id" gorm:"size:32;index"`
	ActorID        string    `json:"actor_id" gorm:"size:96;index"`
	ActorName      string    `json:"actor_name" gorm:"size:160"`
	ActorRole      string    `json:"actor_role" gorm:"size:64;index"`
	GameUsername   string    `json:"game_username" gorm:"size:160"`
	Action         string    `json:"action" gorm:"size:96;index"`
	Method         string    `json:"method" gorm:"size:12"`
	Resource       string    `json:"resource" gorm:"size:200"`
	Server         string    `json:"server" gorm:"size:64"`
	Status         int       `json:"status" gorm:"index"`
	Result         string    `json:"result" gorm:"size:16;index"`
	Error          string    `json:"error,omitempty" gorm:"size:240"`
	DurationMillis int64     `json:"duration_ms"`
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *auditResponseWriter) WriteHeader(status int) {
	if writer.status == 0 {
		writer.status = status
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *auditResponseWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	return writer.ResponseWriter.Write(data)
}

func newAuditRequestID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func shouldAudit(routeName, method string) bool {
	if routeName == "GetAuditEvents" || routeName == "GetSecurityOverview" || routeName == "GenerateMapPreview" {
		return false
	}
	if method != http.MethodGet {
		return managementRoutes[routeName]
	}
	switch routeName {
	case "LogTail", "LoadConfig", "GetServerSettings", "GetPlayerAccess", "ModPortalLoginStatus":
		return true
	default:
		return false
	}
}

func recordAuditEvent(event AuditEvent) {
	if auth.db == nil {
		return
	}
	if err := auth.db.Create(&event).Error; err != nil {
		log.Printf("Could not persist audit event %s", event.RequestID)
		return
	}
	// Retention is time based so the audit database remains bounded without
	// silently discarding recent administrative history.
	_ = auth.db.Where("created_at < ?", time.Now().UTC().Add(-auditRetention)).Delete(&AuditEvent{}).Error
}

func AuditMiddleware(routeName, method string, next http.Handler) http.Handler {
	if !shouldAudit(routeName, method) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := newAuditRequestID()
		w.Header().Set("X-Request-ID", requestID)
		writer := &auditResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		user, _ := r.Context().Value(authContextKey{}).(AuthUser)
		actorName := strings.TrimSpace(user.Name)
		if actorName == "" {
			actorName = user.FactorioUsername()
		}
		if actorName == "" {
			actorName = user.PublicUserID
		}
		result := "success"
		errorText := ""
		if status < 200 || status >= 400 {
			result = "failure"
			errorText = fmt.Sprintf("HTTP %d %s", status, http.StatusText(status))
		}
		recordAuditEvent(AuditEvent{
			CreatedAt: time.Now().UTC(), RequestID: requestID,
			ActorID: user.PublicUserID, ActorName: actorName, ActorRole: user.Role,
			GameUsername: user.FactorioUsername(), Action: routeName, Method: method,
			Resource: notificationResource(r, "Servidor Factorio"), Server: "Factorio",
			Status: status, Result: result, Error: errorText,
			DurationMillis: time.Since(started).Milliseconds(),
		})
	})
}

func GetAuditEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	limit := 100
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > 200 {
		limit = 200
	}
	query := auth.db.Order("id DESC").Limit(limit + 1)
	if before, err := strconv.ParseUint(r.URL.Query().Get("before"), 10, 64); err == nil && before > 0 {
		query = query.Where("id < ?", before)
	}
	if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
		query = query.Where("action = ?", action)
	}
	if result := strings.TrimSpace(r.URL.Query().Get("result")); result == "success" || result == "failure" {
		query = query.Where("result = ?", result)
	}
	events := make([]AuditEvent, 0, limit+1)
	if err := query.Find(&events).Error; err != nil {
		http.Error(w, "Could not read audit history", http.StatusInternalServerError)
		return
	}
	var next uint
	if len(events) > limit {
		next = events[limit-1].ID
		events = events[:limit]
	}
	WriteResponse(w, map[string]interface{}{"events": events, "next_before": next})
}

func GetSecurityOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	user, _ := r.Context().Value(authContextKey{}).(AuthUser)
	protected := make([]string, 0, len(managementRoutes))
	for route := range managementRoutes {
		protected = append(protected, route)
	}
	sort.Strings(protected)
	destructive := make([]string, 0, len(destructiveRoutes))
	for route := range destructiveRoutes {
		destructive = append(destructive, route)
	}
	sort.Strings(destructive)
	WriteResponse(w, map[string]interface{}{
		"role": user.Role, "can_manage": user.CanManage, "server_admin": user.ServerAdmin,
		"management_routes": protected, "destructive_routes": destructive,
		"protections": map[string]bool{"authentication": true, "role_authorization": true, "same_origin_requests": true, "destructive_confirmation": true, "audit_log": true, "secret_redaction": true},
	})
}

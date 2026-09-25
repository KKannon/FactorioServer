package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

const (
	welcomePending = "pending"
	welcomeQueued  = "queued"
	welcomeSent    = "sent"
)

type WelcomeNotification struct {
	PublicUserID string `gorm:"primaryKey;size:64"`
	Recipient    string `gorm:"not null"`
	Name         string `gorm:"not null"`
	Role         string `gorm:"not null"`
	Status       string `gorm:"not null;index"`
	Attempts     int
	CreatedAt    time.Time
	SentAt       *time.Time
}

type notificationPayload struct {
	Event          string `json:"event"`
	Recipient      string `json:"recipient"`
	ActorName      string `json:"actorName"`
	ActorRole      string `json:"actorRole"`
	Resource       string `json:"resource"`
	OccurredAt     string `json:"occurredAt"`
	IdempotencyKey string `json:"idempotencyKey"`
	WelcomeUserID  string `json:"-"`
}

type notificationDispatcher struct {
	endpoint string
	token    string
	client   *http.Client
	queue    chan notificationPayload
	db       *gorm.DB
}

var notifications *notificationDispatcher

var notificationRoutes = map[string]string{
	"StartServer": "Servidor Factorio", "StopServer": "Servidor Factorio",
	"RestartServer": "Servidor Factorio", "KillServer": "Servidor Factorio",
	"CreateWorld": "Mundo Factorio", "CreateSaveBackup": "Save Factorio",
	"RestoreSaveBackup": "Save Factorio", "RemoveSaveBackup": "Save Factorio", "RenameSave": "Save Factorio",
	"AddWhitelistPlayer": "Controle de jogadores", "RemoveWhitelistPlayer": "Controle de jogadores",
	"AddAdmin": "Controle de jogadores", "RemoveAdmin": "Controle de jogadores",
	"AddBan": "Controle de jogadores", "RemoveBan": "Controle de jogadores",
	"UpdateWhitelistPolicy":  "Controle de jogadores",
	"InstallPlayerBridge":    "Inteligência de jogadores",
	"InstallFactorioVersion": "Instalação do Factorio", "UpdateServerSettings": "Configurações do servidor",
}

// SetupNotifications starts a bounded, best-effort delivery worker. Operational
// actions must still succeed if the mail provider is temporarily unavailable.
func SetupNotifications() {
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("STUPID_MAILER_URL")), "/")
	token := strings.TrimSpace(os.Getenv("STUPID_MAIL_INTERNAL_TOKEN"))
	if endpoint == "" && token == "" {
		log.Print("Email notifications are disabled")
		return
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || token == "" {
		log.Print("Email notifications are disabled due to invalid internal configuration")
		return
	}
	notifications = &notificationDispatcher{
		endpoint: endpoint + "/notify",
		token:    token,
		client:   &http.Client{Timeout: 15 * time.Second},
		queue:    make(chan notificationPayload, 64),
		db:       auth.db,
	}
	go notifications.run()
	notifications.recoverWelcomeNotifications()
	log.Print("Email notifications enabled")
}

func (d *notificationDispatcher) run() {
	for payload := range d.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := d.deliver(ctx, payload)
		cancel()
		if err != nil {
			d.completeWelcome(payload, false)
			log.Printf("Email notification delivery failed for event %s: %v", payload.Event, err)
		} else {
			d.completeWelcome(payload, true)
		}
	}
}

func (d *notificationDispatcher) completeWelcome(payload notificationPayload, delivered bool) {
	if payload.WelcomeUserID == "" || d.db == nil {
		return
	}
	updates := map[string]interface{}{"attempts": gorm.Expr("attempts + 1")}
	if delivered {
		now := time.Now().UTC()
		updates["status"] = welcomeSent
		updates["sent_at"] = &now
	} else {
		updates["status"] = welcomePending
	}
	d.db.Model(&WelcomeNotification{}).Where("public_user_id = ?", payload.WelcomeUserID).Updates(updates)
}

func welcomeIdempotencyKey(publicUserID string) string {
	digest := sha256.Sum256([]byte(publicUserID))
	return "factorio/welcome/" + hex.EncodeToString(digest[:16])
}

func (d *notificationDispatcher) queueWelcome(record WelcomeNotification) {
	if d.db == nil {
		return
	}
	claimed := d.db.Model(&WelcomeNotification{}).
		Where("public_user_id = ? AND status = ?", record.PublicUserID, welcomePending).
		Update("status", welcomeQueued)
	if claimed.Error != nil || claimed.RowsAffected != 1 {
		return
	}
	payload := notificationPayload{
		Event: "UserWelcome", Recipient: record.Recipient, ActorName: record.Name, ActorRole: record.Role,
		Resource: "Factorio Server Manager", OccurredAt: record.CreatedAt.UTC().Format(time.RFC3339),
		IdempotencyKey: welcomeIdempotencyKey(record.PublicUserID), WelcomeUserID: record.PublicUserID,
	}
	select {
	case d.queue <- payload:
	default:
		d.db.Model(&WelcomeNotification{}).Where("public_user_id = ?", record.PublicUserID).Update("status", welcomePending)
		log.Print("Email notification queue is full; welcome email remains pending")
	}
}

func (d *notificationDispatcher) recoverWelcomeNotifications() {
	if d.db == nil {
		return
	}
	d.db.Model(&WelcomeNotification{}).Where("status = ?", welcomeQueued).Update("status", welcomePending)
	var pending []WelcomeNotification
	if err := d.db.Where("status = ?", welcomePending).Limit(100).Find(&pending).Error; err != nil {
		log.Print("Could not recover pending welcome notifications")
		return
	}
	for _, record := range pending {
		d.queueWelcome(record)
	}
}

func enqueueWelcomeNotification(user AuthUser) {
	if auth.db == nil || strings.TrimSpace(user.PublicUserID) == "" || strings.TrimSpace(user.Email) == "" {
		return
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Email
	}
	record := WelcomeNotification{
		PublicUserID: user.PublicUserID, Recipient: user.Email, Name: name, Role: user.Role,
		Status: welcomePending, CreatedAt: time.Now().UTC(),
	}
	if err := auth.db.Where("public_user_id = ?", user.PublicUserID).FirstOrCreate(&record).Error; err != nil {
		log.Print("Could not persist welcome notification")
		return
	}
	if notifications != nil && record.Status == welcomePending {
		notifications.queueWelcome(record)
	}
}

func (d *notificationDispatcher) deliver(ctx context.Context, payload notificationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+d.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("internal mailer unavailable")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("internal mailer returned status %d", response.StatusCode)
	}
	return nil
}

func notificationID(routeName string) string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return "factorio/" + routeName + "/" + hex.EncodeToString(random)
	}
	return fmt.Sprintf("factorio/%s/%d", routeName, time.Now().UnixNano())
}

func notificationResource(r *http.Request, fallback string) string {
	variables := mux.Vars(r)
	for _, key := range []string{"save", "backup", "username", "modpack", "mod", "id"} {
		if value := strings.TrimSpace(variables[key]); value != "" {
			value = strings.Map(func(character rune) rune {
				if character < 32 || character == 127 {
					return -1
				}
				return character
			}, value)
			if len(value) > 160 {
				value = value[:160]
			}
			return value
		}
	}
	return fallback
}

func enqueueNotification(routeName string, r *http.Request) {
	if notifications == nil {
		return
	}
	fallback, enabled := notificationRoutes[routeName]
	if !enabled {
		return
	}
	user, ok := r.Context().Value(authContextKey{}).(AuthUser)
	if !ok || strings.TrimSpace(user.Email) == "" {
		return
	}
	actorName := strings.TrimSpace(user.Name)
	if actorName == "" {
		actorName = user.Email
	}
	payload := notificationPayload{
		Event: routeName, Recipient: user.Email, ActorName: actorName, ActorRole: user.Role,
		Resource: notificationResource(r, fallback), OccurredAt: time.Now().UTC().Format(time.RFC3339),
		IdempotencyKey: notificationID(routeName),
	}
	select {
	case notifications.queue <- payload:
	default:
		log.Printf("Email notification queue is full; dropped event %s", routeName)
	}
}

type notificationResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *notificationResponseWriter) WriteHeader(status int) {
	if writer.status == 0 {
		writer.status = status
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *notificationResponseWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	return writer.ResponseWriter.Write(data)
}

func NotificationMiddleware(routeName string, next http.Handler) http.Handler {
	if _, enabled := notificationRoutes[routeName]; !enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &notificationResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= 200 && status < 300 {
			enqueueNotification(routeName, r)
		}
	})
}

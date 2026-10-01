package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type GameEventRule struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"not null"`
	Event           string     `json:"event" gorm:"not null;index"`
	Action          string     `json:"action" gorm:"not null"`
	WebhookURL      string     `json:"webhook_url,omitempty"`
	Template        string     `json:"template"`
	Enabled         bool       `json:"enabled" gorm:"index"`
	CooldownSeconds int        `json:"cooldown_seconds"`
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type GameEventRecord struct {
	ID         uint        `json:"id" gorm:"primaryKey"`
	BridgeID   int64       `json:"bridge_id" gorm:"index"`
	Event      string      `json:"event" gorm:"index"`
	Tick       int64       `json:"tick"`
	Payload    string      `json:"-" gorm:"type:text"`
	OccurredAt time.Time   `json:"occurred_at" gorm:"index"`
	Data       interface{} `json:"data" gorm:"-"`
}

type gameEventDefinition struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	Variables       []string `json:"variables"`
	HighFrequency   bool     `json:"high_frequency"`
	MinimumCooldown int      `json:"minimum_cooldown"`
}

var gameEventCatalog = []gameEventDefinition{
	{Name: "on_entity_damaged", Category: "combat", Description: "An entity took damage.", Variables: []string{"event", "tick", "entity.name", "entity.type", "entity.force", "entity.surface", "entity.position.x", "entity.position.y", "cause.name", "damage_type", "original_damage", "final_damage"}, HighFrequency: true, MinimumCooldown: 5},
	{Name: "on_entity_died", Category: "combat", Description: "An entity died.", Variables: []string{"event", "tick", "entity.name", "entity.type", "entity.force", "entity.surface", "entity.position.x", "entity.position.y", "cause.name", "damage_type"}},
	{Name: "on_player_died", Category: "players", Description: "A player died.", Variables: []string{"event", "tick", "player.name", "player.force", "cause.name"}},
	{Name: "on_player_joined_game", Category: "players", Description: "A player joined the game.", Variables: []string{"event", "tick", "player.name", "player.force"}},
	{Name: "on_player_left_game", Category: "players", Description: "A player left the game.", Variables: []string{"event", "tick", "player.name", "player.force", "reason"}},
	{Name: "on_console_chat", Category: "players", Description: "A player sent a chat message.", Variables: []string{"event", "tick", "player.name", "player.force", "message"}},
	{Name: "on_built_entity", Category: "building", Description: "A player built an entity.", Variables: []string{"event", "tick", "player.name", "entity.name", "entity.type", "entity.surface", "entity.position.x", "entity.position.y"}},
	{Name: "on_player_mined_entity", Category: "building", Description: "A player mined an entity.", Variables: []string{"event", "tick", "player.name", "entity.name", "entity.type", "entity.surface"}},
	{Name: "on_player_crafted_item", Category: "production", Description: "A player crafted an item.", Variables: []string{"event", "tick", "player.name", "item.name", "item.count", "recipe"}, HighFrequency: true, MinimumCooldown: 5},
	{Name: "on_research_finished", Category: "progress", Description: "Research finished.", Variables: []string{"event", "tick", "research.name", "research.level", "research.force"}},
	{Name: "on_rocket_launched", Category: "progress", Description: "A rocket was launched.", Variables: []string{"event", "tick", "rocket.name", "silo.name", "silo.surface"}},
}

var gameEvents struct {
	db         *gorm.DB
	client     *http.Client
	cursor     int64
	configured string
	mu         sync.Mutex
}

func eventDefinition(name string) (gameEventDefinition, bool) {
	for _, definition := range gameEventCatalog {
		if definition.Name == name {
			return definition, true
		}
	}
	return gameEventDefinition{}, false
}

func publicAddress(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}

func validateWebhook(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, errors.New("webhook must be a public HTTPS URL without embedded credentials")
	}
	addresses, err := net.LookupIP(parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("webhook hostname could not be resolved")
	}
	for _, address := range addresses {
		if !publicAddress(address) {
			return nil, errors.New("webhook cannot target local or private networks")
		}
	}
	return parsed, nil
}

func newWebhookClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range addresses {
			if publicAddress(candidate) {
				return dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
			}
		}
		return nil, errors.New("webhook resolved to a non-public address")
	}}
	return &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		_, err := validateWebhook(request.URL.String())
		return err
	}}
}

func SetupGameEvents() {
	if auth.db == nil {
		return
	}
	if err := auth.db.AutoMigrate(&GameEventRule{}, &GameEventRecord{}); err != nil {
		log.Printf("Could not migrate game event tables: %v", err)
		return
	}
	gameEvents.db = auth.db
	gameEvents.client = newWebhookClient()
	go runGameEventListener()
}

func runGameEventListener() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		pollGameEvents()
	}
}

func pollGameEvents() {
	gameEvents.mu.Lock()
	defer gameEvents.mu.Unlock()
	status := factorio.GetFactorioServer().Status()
	if !status.Running || !status.RconConnected {
		gameEvents.configured = ""
		return
	}
	var rules []GameEventRule
	if err := gameEvents.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return
	}
	set := map[string]bool{}
	for _, rule := range rules {
		set[rule.Event] = true
	}
	types := make([]string, 0, len(set))
	for kind := range set {
		types = append(types, kind)
	}
	sort.Strings(types)
	configured := strings.Join(types, ",")
	if configured != gameEvents.configured {
		if err := factorio.ConfigureBridgeEvents(types); err != nil {
			return
		}
		gameEvents.configured = configured
	}
	events, latest, err := factorio.ReadBridgeEvents(gameEvents.cursor)
	if err != nil {
		return
	}
	if latest < gameEvents.cursor {
		gameEvents.cursor = 0
		return
	}
	for _, event := range events {
		persistAndDispatchGameEvent(event, rules)
	}
	if latest > gameEvents.cursor {
		gameEvents.cursor = latest
	}
}

func persistAndDispatchGameEvent(event factorio.GameEvent, rules []GameEventRule) {
	payload := map[string]interface{}{"event": event.Event, "tick": event.Tick}
	for key, value := range event.Payload {
		payload[key] = value
	}
	raw, _ := json.Marshal(payload)
	record := GameEventRecord{BridgeID: event.ID, Event: event.Event, Tick: event.Tick, Payload: string(raw), OccurredAt: time.Now().UTC()}
	_ = gameEvents.db.Create(&record).Error
	if record.ID > 5000 && record.ID%100 == 0 {
		gameEvents.db.Where("id < ?", record.ID-5000).Delete(&GameEventRecord{})
	}
	for index := range rules {
		rule := &rules[index]
		if rule.Event != event.Event {
			continue
		}
		if rule.LastTriggeredAt != nil && time.Since(*rule.LastTriggeredAt) < time.Duration(rule.CooldownSeconds)*time.Second {
			continue
		}
		now := time.Now().UTC()
		rule.LastTriggeredAt = &now
		gameEvents.db.Model(rule).Update("last_triggered_at", now)
		if rule.Action == "webhook" {
			go deliverGameEventWebhook(*rule, payload, now)
		} else if rule.Action == "save_game" {
			_ = factorio.GetFactorioServer().SendRCON("/save")
		}
	}
}

var templateVariable = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

func lookupEventVariable(payload map[string]interface{}, path string) interface{} {
	var value interface{} = payload
	for _, part := range strings.Split(path, ".") {
		object, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		value = object[part]
	}
	return value
}

func renderEventTemplate(template string, payload map[string]interface{}) string {
	return templateVariable.ReplaceAllStringFunc(template, func(token string) string {
		match := templateVariable.FindStringSubmatch(token)
		value := lookupEventVariable(payload, match[1])
		if value == nil {
			return ""
		}
		return fmt.Sprint(value)
	})
}

func deliverGameEventWebhook(rule GameEventRule, payload map[string]interface{}, occurredAt time.Time) {
	if _, err := validateWebhook(rule.WebhookURL); err != nil {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{"rule": rule.Name, "event": payload, "message": renderEventTemplate(rule.Template, payload), "occurred_at": occurredAt.Format(time.RFC3339)})
	request, err := http.NewRequest(http.MethodPost, rule.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Factorio-Server-Manager/Event-Listener")
	response, err := gameEvents.client.Do(request)
	if err != nil {
		log.Printf("Game event webhook %d failed: %v", rule.ID, err)
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Printf("Game event webhook %d returned status %d", rule.ID, response.StatusCode)
	}
}

func validateGameEventRule(rule *GameEventRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Event = strings.TrimSpace(rule.Event)
	rule.Action = strings.TrimSpace(rule.Action)
	rule.Template = strings.TrimSpace(rule.Template)
	definition, ok := eventDefinition(rule.Event)
	if !ok {
		return errors.New("unsupported Factorio event")
	}
	if rule.Name == "" || len(rule.Name) > 100 {
		return errors.New("rule name is required and must be at most 100 characters")
	}
	if rule.Action != "panel" && rule.Action != "webhook" && rule.Action != "save_game" {
		return errors.New("action must be panel, webhook or save_game")
	}
	if rule.Action == "save_game" && rule.CooldownSeconds < 60 {
		rule.CooldownSeconds = 60
	}
	if rule.Action == "webhook" && rule.CooldownSeconds < 1 {
		rule.CooldownSeconds = 1
	}
	if rule.CooldownSeconds < definition.MinimumCooldown {
		rule.CooldownSeconds = definition.MinimumCooldown
	}
	if rule.CooldownSeconds > 86400 {
		return errors.New("cooldown cannot exceed 86400 seconds")
	}
	if len(rule.Template) > 2000 {
		return errors.New("message template is too long")
	}
	if rule.Action == "webhook" {
		parsed, err := validateWebhook(rule.WebhookURL)
		if err != nil {
			return err
		}
		rule.WebhookURL = parsed.String()
	} else {
		rule.WebhookURL = ""
	}
	return nil
}

func GetGameEventCatalog(w http.ResponseWriter, _ *http.Request) {
	WriteResponse(w, map[string]interface{}{"events": gameEventCatalog, "actions": []string{"panel", "webhook", "save_game"}, "bridge": factorio.GetPlayerBridgeStatus()})
}

func ListGameEventRules(w http.ResponseWriter, _ *http.Request) {
	var rules []GameEventRule
	if err := auth.db.Order("id asc").Find(&rules).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	WriteResponse(w, rules)
}

func CreateGameEventRule(w http.ResponseWriter, r *http.Request) {
	var rule GameEventRule
	if resp, err := ReadFromRequestBody(w, r, &rule); err != nil {
		WriteResponse(w, resp)
		return
	}
	if err := validateGameEventRule(&rule); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := auth.db.Create(&rule).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
	WriteResponse(w, rule)
}

func UpdateGameEventRule(w http.ResponseWriter, r *http.Request) {
	var existing GameEventRule
	if err := auth.db.First(&existing, mux.Vars(r)["id"]).Error; err != nil {
		http.Error(w, "rule not found", 404)
		return
	}
	var input GameEventRule
	if resp, err := ReadFromRequestBody(w, r, &input); err != nil {
		WriteResponse(w, resp)
		return
	}
	existing.Name, existing.Event, existing.Action, existing.WebhookURL, existing.Template, existing.Enabled, existing.CooldownSeconds = input.Name, input.Event, input.Action, input.WebhookURL, input.Template, input.Enabled, input.CooldownSeconds
	if err := validateGameEventRule(&existing); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := auth.db.Save(&existing).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	WriteResponse(w, existing)
}

func DeleteGameEventRule(w http.ResponseWriter, r *http.Request) {
	result := auth.db.Delete(&GameEventRule{}, mux.Vars(r)["id"])
	if result.Error != nil {
		http.Error(w, result.Error.Error(), 500)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "rule not found", 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListGameEventHistory(w http.ResponseWriter, _ *http.Request) {
	var records []GameEventRecord
	if err := auth.db.Order("id desc").Limit(200).Find(&records).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for index := range records {
		var data map[string]interface{}
		_ = json.Unmarshal([]byte(records[index].Payload), &data)
		records[index].Data = data
	}
	WriteResponse(w, records)
}

package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var accentColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type UserPreferences struct {
	Theme        string                     `json:"theme"`
	AccentColor  string                     `json:"accent_color"`
	Language     string                     `json:"language"`
	Timezone     string                     `json:"timezone"`
	DateFormat   string                     `json:"date_format"`
	ReduceMotion bool                       `json:"reduce_motion"`
	CompactMode  bool                       `json:"compact_mode"`
	Extra        map[string]json.RawMessage `json:"-"`
}

func (p *UserPreferences) UnmarshalJSON(data []byte) error {
	type alias UserPreferences
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, key := range []string{"theme", "accent_color", "language", "timezone", "date_format", "reduce_motion", "compact_mode"} {
		delete(all, key)
	}
	*p = UserPreferences(known)
	p.Extra = all
	p.normalize()
	return nil
}

func (p UserPreferences) MarshalJSON() ([]byte, error) {
	values := make(map[string]json.RawMessage, len(p.Extra)+7)
	for key, value := range p.Extra {
		values[key] = value
	}
	known := map[string]interface{}{"theme": p.Theme, "accent_color": p.AccentColor, "language": p.Language, "timezone": p.Timezone, "date_format": p.DateFormat, "reduce_motion": p.ReduceMotion, "compact_mode": p.CompactMode}
	for key, value := range known {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		values[key] = raw
	}
	return json.Marshal(values)
}

func (p *UserPreferences) normalize() {
	if p.Theme != "system" && p.Theme != "light" && p.Theme != "dark" {
		p.Theme = "system"
	}
	if !accentColorPattern.MatchString(p.AccentColor) {
		p.AccentColor = "#E39827"
	} else {
		p.AccentColor = strings.ToUpper(p.AccentColor)
	}
	if p.Language == "" {
		p.Language = "pt-BR"
	}
	if p.Timezone == "" {
		p.Timezone = "America/Sao_Paulo"
	}
	if p.DateFormat != "DD/MM/YYYY" && p.DateFormat != "MM/DD/YYYY" && p.DateFormat != "YYYY-MM-DD" {
		p.DateFormat = "DD/MM/YYYY"
	}
}

// AuthUser is the sole application-facing identity adapter. Tokens and raw
// provider payloads never leave the backend authentication layer.
type AuthUser struct {
	Subject      string            `json:"sub"`
	PublicUserID string            `json:"public_user_id"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	GivenName    string            `json:"given_name"`
	FamilyName   string            `json:"family_name"`
	Picture      string            `json:"picture,omitempty"`
	Username     string            `json:"username,omitempty"`
	PreferredUsername string       `json:"preferred_username,omitempty"`
	GameUsername string            `json:"game_username,omitempty"`
	CanManage   bool               `json:"can_manage"`
	OriginApp    string            `json:"origin_app"`
	Apps         []string          `json:"apps"`
	Role         string            `json:"role"`
	Roles        map[string]string `json:"roles"`
	Preferences  UserPreferences   `json:"preferences"`
}

func (u AuthUser) FactorioUsername() string {
	for _, candidate := range []string{u.GameUsername, u.PreferredUsername, u.Username} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	if local, _, ok := strings.Cut(u.Email, "@"); ok {
		return local
	}
	return ""
}

func (u AuthUser) Validate(appSlug string) error {
	if u.Subject == "" || u.PublicUserID == "" {
		return fmt.Errorf("userinfo is missing a stable identity")
	}
	if u.Role == "" {
		return fmt.Errorf("userinfo does not grant a role for this application")
	}
	linked := false
	for _, app := range u.Apps {
		if app == appSlug {
			linked = true
			break
		}
	}
	if role, ok := u.Roles[appSlug]; ok && role != "" {
		linked = true
	}
	if !linked {
		return fmt.Errorf("userinfo does not link the user to this application")
	}
	if role, ok := u.Roles[appSlug]; ok && role != "" && role != u.Role {
		return fmt.Errorf("userinfo role is inconsistent for this application")
	}
	return nil
}

func (u AuthUser) Initials() string {
	parts := strings.Fields(strings.TrimSpace(u.Name))
	if len(parts) == 0 {
		parts = strings.Fields(strings.TrimSpace(u.Email))
	}
	if len(parts) == 0 {
		return "?"
	}
	initials := strings.ToUpper(string([]rune(parts[0])[0]))
	if len(parts) > 1 {
		initials += strings.ToUpper(string([]rune(parts[len(parts)-1])[0]))
	}
	return initials
}

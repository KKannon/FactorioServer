package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type factorioClientConfig struct {
	PublicGameHost      string `json:"public_game_host"`
	GamePort            int    `json:"game_port"`
	GameLanguage        string `json:"game_language"`
	Fullscreen          bool   `json:"fullscreen"`
	WindowSize          string `json:"window_size"`
	GraphicsPreset      string `json:"graphics_preset"`
	DisableAudio        bool   `json:"disable_audio"`
	OfficialDownloadURL string `json:"official_download_url"`
}

var factorioClientConfigMu sync.Mutex
var windowSizePattern = regexp.MustCompile(`^(maximized|[1-9][0-9]{2,4}x[1-9][0-9]{2,4})$`)
var hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

func defaultFactorioClientConfig() factorioClientConfig {
	return factorioClientConfig{GamePort: 34197, GameLanguage: "pt-BR", Fullscreen: true, WindowSize: "maximized", GraphicsPreset: "high", OfficialDownloadURL: "https://factorio.com/download"}
}

func factorioClientConfigPath() string {
	return filepath.Join(filepath.Dir(bootstrap.GetConfig().ConfFile), "factorio-client.json")
}

func validateFactorioClientConfig(config factorioClientConfig) error {
	if config.GamePort < 1 || config.GamePort > 65535 {
		return errors.New("game_port must be between 1 and 65535")
	}
	if config.PublicGameHost != "" && (strings.ContainsAny(config.PublicGameHost, "/\\") || net.ParseIP(config.PublicGameHost) == nil && !hostnamePattern.MatchString(config.PublicGameHost)) {
		return errors.New("public_game_host must be a hostname or IP address")
	}
	allowedLanguages := map[string]bool{"en": true, "pt-BR": true, "es-ES": true}
	if !allowedLanguages[config.GameLanguage] {
		return errors.New("game_language must be en, pt-BR, or es-ES")
	}
	if !windowSizePattern.MatchString(config.WindowSize) {
		return errors.New("window_size must be maximized or WIDTHxHEIGHT")
	}
	allowedPresets := map[string]bool{"very-low": true, "low": true, "medium": true, "high": true, "very-high": true, "extreme": true}
	if !allowedPresets[config.GraphicsPreset] {
		return errors.New("graphics_preset is not supported")
	}
	downloadURL, err := url.Parse(config.OfficialDownloadURL)
	if err != nil || downloadURL.Scheme != "https" || downloadURL.Host == "" {
		return errors.New("official_download_url must be a valid HTTPS URL")
	}
	return nil
}

func loadFactorioClientConfig() (factorioClientConfig, error) {
	factorioClientConfigMu.Lock()
	defer factorioClientConfigMu.Unlock()
	config := defaultFactorioClientConfig()
	data, err := os.ReadFile(factorioClientConfigPath())
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err = json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, validateFactorioClientConfig(config)
}

func saveFactorioClientConfig(config factorioClientConfig) error {
	if err := validateFactorioClientConfig(config); err != nil {
		return err
	}
	factorioClientConfigMu.Lock()
	defer factorioClientConfigMu.Unlock()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	path := factorioClientConfigPath()
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func GetFactorioClientConfig(w http.ResponseWriter, _ *http.Request) {
	config, err := loadFactorioClientConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(config)
}

func UpdateFactorioClientConfig(w http.ResponseWriter, r *http.Request) {
	var config factorioClientConfig
	if _, err := ReadFromRequestBody(w, r, &config); err != nil {
		return
	}
	if err := saveFactorioClientConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(config)
}

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
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

type factorioClientConfig struct {
	PublicGameHost            string `json:"public_game_host"`
	GamePort                  int    `json:"game_port"`
	IncludeGamePort           bool   `json:"include_game_port"`
	OfficialDownloadURL       string `json:"official_download_url"`
	AlternativeDownloadMagnet string `json:"alternative_download_magnet"`
}

var factorioClientConfigMu sync.Mutex
var hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

func defaultFactorioClientConfig() factorioClientConfig {
	return factorioClientConfig{GamePort: 34197, IncludeGamePort: true, OfficialDownloadURL: "https://factorio.com/download"}
}

func factorioClientConfigPath() string {
	return filepath.Join(filepath.Dir(bootstrap.GetConfig().ConfFile), "factorio-client.json")
}

func validateFactorioClientConfig(config factorioClientConfig) error {
	if config.IncludeGamePort && (config.GamePort < 1 || config.GamePort > 65535) {
		return errors.New("game_port must be between 1 and 65535")
	}
	if !config.IncludeGamePort && config.GamePort != 0 && (config.GamePort < 1 || config.GamePort > 65535) {
		return errors.New("game_port must be empty or between 1 and 65535")
	}
	if config.PublicGameHost != "" && (strings.ContainsAny(config.PublicGameHost, "/\\") || net.ParseIP(config.PublicGameHost) == nil && !hostnamePattern.MatchString(config.PublicGameHost)) {
		return errors.New("public_game_host must be a hostname or IP address")
	}
	downloadURL, err := url.Parse(config.OfficialDownloadURL)
	if err != nil || downloadURL.Scheme != "https" || downloadURL.Host == "" {
		return errors.New("official_download_url must be a valid HTTPS URL")
	}
	if config.AlternativeDownloadMagnet != "" {
		magnet, magnetErr := url.Parse(config.AlternativeDownloadMagnet)
		exactTopic := strings.ToLower(magnet.Query().Get("xt"))
		if magnetErr != nil || !strings.EqualFold(magnet.Scheme, "magnet") || !strings.HasPrefix(exactTopic, "urn:bt") {
			return errors.New("alternative_download_magnet must be empty or a valid magnet link")
		}
	}
	return nil
}

func allowsUnauthenticatedPlayers() bool {
	required, ok := factorio.GetFactorioServer().Settings["require_user_verification"].(bool)
	return ok && !required
}

func effectiveFactorioClientConfig(config factorioClientConfig, allowUnauthenticated bool) factorioClientConfig {
	if !allowUnauthenticated {
		config.AlternativeDownloadMagnet = ""
	}
	return config
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
	if config.AlternativeDownloadMagnet != "" && !allowsUnauthenticatedPlayers() {
		http.Error(w, "alternative_download_magnet requires unauthenticated players to be allowed", http.StatusBadRequest)
		return
	}
	if err := saveFactorioClientConfig(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(config)
}

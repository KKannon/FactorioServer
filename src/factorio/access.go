package factorio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

var (
	accessFileMutex   = sync.Mutex{}
	playerNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{2,64}$`)
)

func ValidPlayerName(value string) bool {
	return playerNamePattern.MatchString(strings.TrimSpace(value))
}

func readAccessList(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func addAccessEntry(path, username string) (bool, error) {
	values, err := readAccessList(path)
	if err != nil {
		return false, err
	}
	for _, value := range values {
		if strings.EqualFold(value, username) {
			return false, nil
		}
	}
	values = append(values, username)
	sort.Strings(values)
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, 0664)
}

func removeAccessEntry(path, username string) (bool, error) {
	values, err := readAccessList(path)
	if err != nil {
		return false, err
	}
	filtered := make([]string, 0, len(values))
	removed := false
	for _, value := range values {
		if strings.EqualFold(value, username) {
			removed = true
			continue
		}
		filtered = append(filtered, value)
	}
	if !removed {
		return false, nil
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, 0664)
}

// EnsurePlayerAccess mirrors application access into Factorio's whitelist and,
// for management roles, into its administrator list.
func EnsurePlayerAccess(username string, administrator bool) error {
	username = strings.TrimSpace(username)
	if !ValidPlayerName(username) {
		return fmt.Errorf("invalid Factorio username %q", username)
	}
	config := bootstrap.GetConfig()
	if config.FactorioWhitelistFile == "" || config.FactorioAdminFile == "" {
		return fmt.Errorf("Factorio access files are not configured")
	}

	accessFileMutex.Lock()
	whitelistAdded, err := addAccessEntry(config.FactorioWhitelistFile, username)
	adminAdded, adminRemoved := false, false
	if err == nil {
		if administrator {
			adminAdded, err = addAccessEntry(config.FactorioAdminFile, username)
		} else {
			adminRemoved, err = removeAccessEntry(config.FactorioAdminFile, username)
		}
	}
	accessFileMutex.Unlock()
	if err != nil {
		return err
	}

	server := GetFactorioServer()
	if server.GetRunning() && server.Rcon != nil {
		if whitelistAdded {
			_, _ = server.Rcon.Write("/whitelist add " + username)
		}
		if adminAdded {
			_, _ = server.Rcon.Write("/promote " + username)
		}
		if adminRemoved {
			_, _ = server.Rcon.Write("/demote " + username)
		}
	}
	return nil
}

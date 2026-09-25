package factorio

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type BanEntry struct {
	Username string `json:"username,omitempty"`
	Address  string `json:"address,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type AccessOverview struct {
	WhitelistEnabled bool       `json:"whitelist_enabled"`
	Whitelist        []string   `json:"whitelist"`
	Admins           []string   `json:"admins"`
	Bans             []BanEntry `json:"bans"`
}

type accessPolicy struct {
	WhitelistEnabled bool `json:"whitelist_enabled"`
}

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

func writeJSONFile(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0664)
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
	return true, writeJSONFile(path, values)
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
	return true, writeJSONFile(path, filtered)
}

func accessPolicyPath() string {
	return filepath.Join(bootstrap.GetConfig().FactorioConfigDir, "fsm-access-policy.json")
}

func readWhitelistEnabledAt(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	policy := accessPolicy{WhitelistEnabled: true}
	if err := json.Unmarshal(data, &policy); err != nil {
		return true, err
	}
	return policy.WhitelistEnabled, nil
}

func readWhitelistEnabled() (bool, error) {
	return readWhitelistEnabledAt(accessPolicyPath())
}

// WhitelistEnabled returns the persisted runtime policy. Invalid or missing
// policy data fails closed, keeping the whitelist enabled.
func WhitelistEnabled() bool {
	enabled, err := readWhitelistEnabled()
	if err != nil {
		log.Printf("could not read whitelist policy; keeping it enabled: %v", err)
		return true
	}
	return enabled
}

func readBanRecords(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	var values []json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func listBans(path string) ([]BanEntry, error) {
	records, err := readBanRecords(path)
	if err != nil {
		return nil, err
	}
	entries := make([]BanEntry, 0, len(records))
	for _, record := range records {
		var entry BanEntry
		if err := json.Unmarshal(record, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func syncAccessCommand(command string) error {
	server := GetFactorioServer()
	if !server.GetRunning() {
		return nil
	}
	if !server.Status().RconConnected {
		return fmt.Errorf("Factorio is running but RCON is not connected")
	}
	if err := server.SendRCON(command); err != nil {
		return fmt.Errorf("apply access change through RCON: %w", err)
	}
	return nil
}

func ListAccessOverview() (AccessOverview, error) {
	accessFileMutex.Lock()
	defer accessFileMutex.Unlock()
	config := bootstrap.GetConfig()
	whitelist, err := readAccessList(config.FactorioWhitelistFile)
	if err != nil {
		return AccessOverview{}, err
	}
	admins, err := readAccessList(config.FactorioAdminFile)
	if err != nil {
		return AccessOverview{}, err
	}
	bans, err := listBans(config.FactorioBanFile)
	if err != nil {
		return AccessOverview{}, err
	}
	enabled, err := readWhitelistEnabled()
	if err != nil {
		return AccessOverview{}, err
	}
	return AccessOverview{WhitelistEnabled: enabled, Whitelist: whitelist, Admins: admins, Bans: bans}, nil
}

func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if !ValidPlayerName(username) {
		return "", fmt.Errorf("invalid Factorio username %q", username)
	}
	return username, nil
}

func changeStringAccessList(path, username, command string, add bool) error {
	username, err := validateUsername(username)
	if err != nil {
		return err
	}
	accessFileMutex.Lock()
	defer accessFileMutex.Unlock()
	original, err := readAccessList(path)
	if err != nil {
		return err
	}
	changed := false
	if add {
		changed, err = addAccessEntry(path, username)
	} else {
		changed, err = removeAccessEntry(path, username)
	}
	if err != nil || !changed {
		return err
	}
	if err := syncAccessCommand(command + username); err != nil {
		if rollbackErr := writeJSONFile(path, original); rollbackErr != nil {
			return fmt.Errorf("%v; rollback failed: %w", err, rollbackErr)
		}
		return err
	}
	return nil
}

func AddWhitelistPlayer(username string) error {
	return changeStringAccessList(bootstrap.GetConfig().FactorioWhitelistFile, username, "/whitelist add ", true)
}

func RemoveWhitelistPlayer(username string) error {
	return changeStringAccessList(bootstrap.GetConfig().FactorioWhitelistFile, username, "/whitelist remove ", false)
}

func AddAdmin(username string) error {
	return changeStringAccessList(bootstrap.GetConfig().FactorioAdminFile, username, "/promote ", true)
}

func RemoveAdmin(username string) error {
	return changeStringAccessList(bootstrap.GetConfig().FactorioAdminFile, username, "/demote ", false)
}

func SetWhitelistEnabled(enabled bool) error {
	accessFileMutex.Lock()
	defer accessFileMutex.Unlock()
	previous, err := readWhitelistEnabled()
	if err != nil {
		return err
	}
	if previous == enabled {
		return nil
	}
	if err := writeJSONFile(accessPolicyPath(), accessPolicy{WhitelistEnabled: enabled}); err != nil {
		return err
	}
	command := "/whitelist disable"
	if enabled {
		command = "/whitelist enable"
	}
	if err := syncAccessCommand(command); err != nil {
		if rollbackErr := writeJSONFile(accessPolicyPath(), accessPolicy{WhitelistEnabled: previous}); rollbackErr != nil {
			return fmt.Errorf("%v; rollback failed: %w", err, rollbackErr)
		}
		return err
	}
	return nil
}

func validBanReason(reason string) bool {
	if len(reason) > 500 {
		return false
	}
	for _, character := range reason {
		if character == 0 || unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func AddBan(username, reason string) error {
	username, err := validateUsername(username)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if !validBanReason(reason) {
		return fmt.Errorf("ban reason is invalid or longer than 500 characters")
	}
	accessFileMutex.Lock()
	defer accessFileMutex.Unlock()
	path := bootstrap.GetConfig().FactorioBanFile
	records, err := readBanRecords(path)
	if err != nil {
		return err
	}
	for _, record := range records {
		var entry BanEntry
		if err := json.Unmarshal(record, &entry); err != nil {
			return err
		}
		if strings.EqualFold(entry.Username, username) {
			return nil
		}
	}
	encoded, err := json.Marshal(BanEntry{Username: username, Reason: reason})
	if err != nil {
		return err
	}
	updated := append(append([]json.RawMessage{}, records...), encoded)
	if err := writeJSONFile(path, updated); err != nil {
		return err
	}
	command := "/ban " + username
	if reason != "" {
		command += " " + reason
	}
	if err := syncAccessCommand(command); err != nil {
		if rollbackErr := writeJSONFile(path, records); rollbackErr != nil {
			return fmt.Errorf("%v; rollback failed: %w", err, rollbackErr)
		}
		return err
	}
	return nil
}

func RemoveBan(username string) error {
	username, err := validateUsername(username)
	if err != nil {
		return err
	}
	accessFileMutex.Lock()
	defer accessFileMutex.Unlock()
	path := bootstrap.GetConfig().FactorioBanFile
	records, err := readBanRecords(path)
	if err != nil {
		return err
	}
	updated := make([]json.RawMessage, 0, len(records))
	removed := false
	for _, record := range records {
		var entry BanEntry
		if err := json.Unmarshal(record, &entry); err != nil {
			return err
		}
		if strings.EqualFold(entry.Username, username) {
			removed = true
			continue
		}
		updated = append(updated, record)
	}
	if !removed {
		return nil
	}
	if err := writeJSONFile(path, updated); err != nil {
		return err
	}
	if err := syncAccessCommand("/unban " + username); err != nil {
		if rollbackErr := writeJSONFile(path, records); rollbackErr != nil {
			return fmt.Errorf("%v; rollback failed: %w", err, rollbackErr)
		}
		return err
	}
	return nil
}

// EnsurePlayerAccess mirrors application access into Factorio's whitelist and,
// for configured Factorio administrator roles, into its administrator list.
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
	if server.GetRunning() && server.Status().RconConnected {
		if whitelistAdded {
			_ = server.SendRCON("/whitelist add " + username)
		}
		if adminAdded {
			_ = server.SendRCON("/promote " + username)
		}
		if adminRemoved {
			_ = server.SendRCON("/demote " + username)
		}
	}
	return nil
}

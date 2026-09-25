package factorio

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/rcon"
)

const (
	PlayerBridgeName     = "factorio-server-manager-bridge"
	playerBridgeVersion  = "1.0.0"
	playerBridgeResponse = "FSM_PLAYER_INTELLIGENCE:"
)

//go:embed player_bridge/*
var playerBridgeFiles embed.FS

type PlayerItem struct {
	Name    string `json:"name"`
	Quality string `json:"quality"`
	Count   int    `json:"count"`
}

type PlayerEquipment struct {
	Name    string  `json:"name"`
	Quality string  `json:"quality"`
	Shield  float64 `json:"shield"`
	Energy  float64 `json:"energy"`
}

type PlayerPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PlayerCraft struct {
	Recipe       string `json:"recipe"`
	Count        int    `json:"count"`
	Prerequisite bool   `json:"prerequisite"`
}

type PlayerTrackedStatistics struct {
	Scope        string  `json:"scope"`
	Deaths       int     `json:"deaths"`
	CraftedItems int     `json:"crafted_items"`
	Distance     float64 `json:"distance"`
}

type PlayerIntelligence struct {
	Name            string                  `json:"name"`
	Connected       bool                    `json:"connected"`
	Admin           bool                    `json:"admin"`
	Force           string                  `json:"force"`
	PermissionGroup *string                 `json:"permission_group"`
	Surface         *string                 `json:"surface"`
	Position        *PlayerPosition         `json:"position"`
	OnlineTicks     int64                   `json:"online_ticks"`
	LastOnlineTick  int64                   `json:"last_online_tick"`
	AFKTicks        int64                   `json:"afk_ticks"`
	Health          *float64                `json:"health"`
	MaxHealth       *float64                `json:"max_health"`
	Inventory       []PlayerItem            `json:"inventory"`
	Guns            []PlayerItem            `json:"guns"`
	Ammo            []PlayerItem            `json:"ammo"`
	Armor           []PlayerItem            `json:"armor"`
	Trash           []PlayerItem            `json:"trash"`
	Equipment       []PlayerEquipment       `json:"equipment"`
	CraftingQueue   []PlayerCraft           `json:"crafting_queue"`
	Statistics      PlayerTrackedStatistics `json:"statistics"`
}

type PlayerCapabilities struct {
	Inventory          bool   `json:"inventory"`
	Position           bool   `json:"position"`
	Playtime           bool   `json:"playtime"`
	Equipment          bool   `json:"equipment"`
	CraftingQueue      bool   `json:"crafting_queue"`
	Health             bool   `json:"health"`
	TrackedStatistics  bool   `json:"tracked_statistics"`
	Achievements       bool   `json:"achievements"`
	AchievementsReason string `json:"achievements_reason"`
}

type PlayerIntelligenceSnapshot struct {
	SchemaVersion int                  `json:"schema_version"`
	GeneratedTick int64                `json:"generated_tick"`
	Source        string               `json:"source"`
	Players       []PlayerIntelligence `json:"players"`
	Capabilities  PlayerCapabilities   `json:"capabilities"`
}

type PlayerBridgeStatus struct {
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
	Version   string `json:"version,omitempty"`
}

func buildPlayerBridgeArchive() ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	prefix := PlayerBridgeName + "_" + playerBridgeVersion + "/"
	err := fs.WalkDir(playerBridgeFiles, "player_bridge", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		content, err := playerBridgeFiles.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := writer.Create(prefix + filepath.Base(path))
		if err != nil {
			return err
		}
		_, err = file.Write(content)
		return err
	})
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func InstallPlayerBridge() error {
	config := bootstrap.GetConfig()
	if err := os.MkdirAll(config.FactorioModsDir, 0755); err != nil {
		return err
	}
	archive, err := buildPlayerBridgeArchive()
	if err != nil {
		return fmt.Errorf("build player bridge: %w", err)
	}
	mods, err := NewMods(config.FactorioModsDir)
	if err != nil {
		return err
	}
	filename := PlayerBridgeName + "_" + playerBridgeVersion + ".zip"
	if err := mods.createMod(PlayerBridgeName, filename, bytes.NewReader(archive)); err != nil {
		return fmt.Errorf("install player bridge: %w", err)
	}
	return nil
}

func GetPlayerBridgeStatus() PlayerBridgeStatus {
	status := PlayerBridgeStatus{}
	mods, err := NewMods(bootstrap.GetConfig().FactorioModsDir)
	if err != nil {
		return status
	}
	for _, installed := range mods.ListInstalledMods().ModsResult {
		if installed.Name == PlayerBridgeName {
			status.Installed = true
			status.Enabled = installed.Enabled
			status.Version = installed.Version
			return status
		}
	}
	return status
}

func rconAddress() string {
	config := bootstrap.GetConfig()
	host := strings.TrimSpace(config.ServerIP)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(config.FactorioRconPort))
}

func requestPlayerSnapshot() (string, error) {
	config := bootstrap.GetConfig()
	console, err := rcon.Dial(rconAddress(), config.FactorioRconPass)
	if err != nil {
		return "", fmt.Errorf("connect to Factorio RCON: %w", err)
	}
	defer console.Close()
	requestID, err := console.Write("/fsm-players")
	if err != nil {
		return "", fmt.Errorf("request player snapshot: %w", err)
	}
	response, responseID, err := console.Read()
	if err != nil {
		return "", fmt.Errorf("read player snapshot: %w", err)
	}
	if requestID != responseID {
		return "", errors.New("unexpected RCON response identifier")
	}
	return response, nil
}

func parsePlayerSnapshot(response string) (PlayerIntelligenceSnapshot, error) {
	var snapshot PlayerIntelligenceSnapshot
	index := strings.Index(response, playerBridgeResponse)
	if index < 0 {
		return snapshot, errors.New("player intelligence bridge did not answer")
	}
	payload := strings.TrimSpace(response[index+len(playerBridgeResponse):])
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode player intelligence: %w", err)
	}
	if snapshot.SchemaVersion != 1 || snapshot.Source != PlayerBridgeName {
		return snapshot, errors.New("unsupported player intelligence response")
	}
	return snapshot, nil
}

func ReadPlayerIntelligence() (PlayerIntelligenceSnapshot, error) {
	server := GetFactorioServer()
	status := server.Status()
	if !status.Running {
		return PlayerIntelligenceSnapshot{}, errors.New("Factorio is not running")
	}
	if !status.RconConnected {
		return PlayerIntelligenceSnapshot{}, errors.New("Factorio RCON is not connected")
	}
	response, err := requestPlayerSnapshot()
	if err != nil {
		return PlayerIntelligenceSnapshot{}, err
	}
	return parsePlayerSnapshot(response)
}

func VisiblePlayerIntelligence(snapshot PlayerIntelligenceSnapshot, username string, allPlayers bool) []PlayerIntelligence {
	if allPlayers {
		return snapshot.Players
	}
	visible := make([]PlayerIntelligence, 0, 1)
	for _, player := range snapshot.Players {
		if player.Name == username {
			visible = append(visible, player)
			break
		}
	}
	return visible
}

package factorio

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

const (
	PlayerBridgeName     = "factorio-server-manager-bridge"
	playerBridgeVersion  = "1.1.0"
	playerBridgeResponse = "FSM_PLAYER_INTELLIGENCE:"
	maxRCONPacketSize    = 16 << 20
)

const (
	rconResponsePacket = int32(0)
	rconExecPacket     = int32(2)
	rconAuthPacket     = int32(3)
	rconAuthResponse   = int32(2)
)

type rconPacket struct {
	id         int32
	packetType int32
	payload    string
}

//go:embed player_bridge/*
var playerBridgeFiles embed.FS

type PlayerItem struct {
	Name    string `json:"name"`
	Quality string `json:"quality"`
	Count   int    `json:"count"`
}

type PlayerItemList []PlayerItem

type PlayerEquipment struct {
	Name    string  `json:"name"`
	Quality string  `json:"quality"`
	Shield  float64 `json:"shield"`
	Energy  float64 `json:"energy"`
}

type PlayerEquipmentList []PlayerEquipment

type PlayerPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PlayerCraft struct {
	Recipe       string `json:"recipe"`
	Count        int    `json:"count"`
	Prerequisite bool   `json:"prerequisite"`
}

type PlayerCraftList []PlayerCraft

func unmarshalPlayerList[T any](data []byte, target *[]T) error {
	var values []T
	if err := json.Unmarshal(data, &values); err == nil {
		*target = values
		return nil
	}
	var emptyObject map[string]json.RawMessage
	if err := json.Unmarshal(data, &emptyObject); err == nil && len(emptyObject) == 0 {
		*target = make([]T, 0)
		return nil
	}
	return json.Unmarshal(data, &values)
}

func (items *PlayerItemList) UnmarshalJSON(data []byte) error {
	return unmarshalPlayerList(data, (*[]PlayerItem)(items))
}

func (items *PlayerEquipmentList) UnmarshalJSON(data []byte) error {
	return unmarshalPlayerList(data, (*[]PlayerEquipment)(items))
}

func (items *PlayerCraftList) UnmarshalJSON(data []byte) error {
	return unmarshalPlayerList(data, (*[]PlayerCraft)(items))
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
	Inventory       PlayerItemList          `json:"inventory"`
	Guns            PlayerItemList          `json:"guns"`
	Ammo            PlayerItemList          `json:"ammo"`
	Armor           PlayerItemList          `json:"armor"`
	Trash           PlayerItemList          `json:"trash"`
	Equipment       PlayerEquipmentList     `json:"equipment"`
	CraftingQueue   PlayerCraftList         `json:"crafting_queue"`
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
	Installed        bool   `json:"installed"`
	Enabled          bool   `json:"enabled"`
	Version          string `json:"version,omitempty"`
	AvailableVersion string `json:"available_version"`
	UpdateAvailable  bool   `json:"update_available"`
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

// PlayerBridgeArchive returns the exact private mod archive used by the
// server. Clients need this file because the bridge is not published on the
// Factorio mod portal and therefore cannot be synchronized automatically.
func PlayerBridgeArchive() ([]byte, string, error) {
	archive, err := buildPlayerBridgeArchive()
	if err != nil {
		return nil, "", err
	}
	return archive, PlayerBridgeName + "_" + playerBridgeVersion + ".zip", nil
}

func InstallPlayerBridge() error {
	config := bootstrap.GetConfig()
	if err := os.MkdirAll(config.FactorioModsDir, 0755); err != nil {
		return err
	}
	archive, filename, err := PlayerBridgeArchive()
	if err != nil {
		return fmt.Errorf("build player bridge: %w", err)
	}
	mods, err := NewMods(config.FactorioModsDir)
	if err != nil {
		return err
	}
	if err := mods.createMod(PlayerBridgeName, filename, bytes.NewReader(archive)); err != nil {
		return fmt.Errorf("install player bridge: %w", err)
	}
	return nil
}

// UpdateInstalledPlayerBridge keeps the bundled private bridge compatible with
// the manager release. It does not install the bridge for servers that have
// never opted into player intelligence.
func UpdateInstalledPlayerBridge() (bool, error) {
	status := GetPlayerBridgeStatus()
	if !status.Installed || !status.UpdateAvailable {
		return false, nil
	}
	if err := InstallPlayerBridge(); err != nil {
		return false, err
	}
	return true, nil
}

func GetPlayerBridgeStatus() PlayerBridgeStatus {
	status := PlayerBridgeStatus{AvailableVersion: playerBridgeVersion}
	mods, err := NewMods(bootstrap.GetConfig().FactorioModsDir)
	if err != nil {
		return status
	}
	for _, installed := range mods.ListInstalledMods().ModsResult {
		if installed.Name == PlayerBridgeName {
			status.Installed = true
			status.Enabled = installed.Enabled
			status.Version = installed.Version
			status.UpdateAvailable = installed.Version != playerBridgeVersion
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

func writeRCONPacket(writer io.Writer, id int32, packetType int32, payload string) error {
	if len(payload)+10 > maxRCONPacketSize {
		return errors.New("RCON request exceeds the safe packet limit")
	}
	packet := bytes.NewBuffer(make([]byte, 0, len(payload)+14))
	if err := binary.Write(packet, binary.LittleEndian, int32(len(payload)+10)); err != nil {
		return err
	}
	if err := binary.Write(packet, binary.LittleEndian, id); err != nil {
		return err
	}
	if err := binary.Write(packet, binary.LittleEndian, packetType); err != nil {
		return err
	}
	_, _ = packet.WriteString(payload)
	_ = packet.WriteByte(0)
	_ = packet.WriteByte(0)
	_, err := io.Copy(writer, packet)
	return err
}

func readRCONPacket(reader io.Reader) (rconPacket, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return rconPacket{}, err
	}
	size := int64(int32(binary.LittleEndian.Uint32(header[:])))
	if size < 10 || size > maxRCONPacketSize {
		return rconPacket{}, fmt.Errorf("invalid RCON packet size %d", size)
	}
	body := make([]byte, int(size))
	if _, err := io.ReadFull(reader, body); err != nil {
		return rconPacket{}, err
	}
	if body[len(body)-2] != 0 || body[len(body)-1] != 0 {
		return rconPacket{}, errors.New("invalid RCON packet terminator")
	}
	return rconPacket{
		id:         int32(binary.LittleEndian.Uint32(body[0:4])),
		packetType: int32(binary.LittleEndian.Uint32(body[4:8])),
		payload:    string(body[8 : len(body)-2]),
	}, nil
}

func requestBridgeCommand(command string) (string, error) {
	config := bootstrap.GetConfig()
	console, err := net.DialTimeout("tcp", rconAddress(), 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("connect to Factorio RCON: %w", err)
	}
	defer console.Close()
	if err := console.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", fmt.Errorf("set Factorio RCON deadline: %w", err)
	}
	const authID = int32(1)
	if err := writeRCONPacket(console, authID, rconAuthPacket, config.FactorioRconPass); err != nil {
		return "", fmt.Errorf("authenticate with Factorio RCON: %w", err)
	}
	authenticated := false
	for attempts := 0; attempts < 2; attempts++ {
		response, readErr := readRCONPacket(console)
		if readErr != nil {
			return "", fmt.Errorf("read Factorio RCON authentication: %w", readErr)
		}
		if response.packetType != rconAuthResponse {
			continue
		}
		if response.id == -1 {
			return "", errors.New("Factorio RCON authentication failed")
		}
		if response.id != authID {
			return "", errors.New("unexpected Factorio RCON authentication identifier")
		}
		authenticated = true
		break
	}
	if !authenticated {
		return "", errors.New("Factorio RCON did not confirm authentication")
	}

	const requestID = int32(2)
	if err := writeRCONPacket(console, requestID, rconExecPacket, command); err != nil {
		return "", fmt.Errorf("request bridge command: %w", err)
	}
	for attempts := 0; attempts < 4; attempts++ {
		response, readErr := readRCONPacket(console)
		if readErr != nil {
			return "", fmt.Errorf("read bridge response: %w", readErr)
		}
		if response.id == requestID && response.packetType == rconResponsePacket {
			return response.payload, nil
		}
	}
	return "", errors.New("Factorio RCON did not return the requested bridge response")
}

func requestPlayerSnapshot() (string, error) { return requestBridgeCommand("/fsm-players") }

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

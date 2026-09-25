package factorio

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

var nativeMapPresets = []string{"default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "lakes", "island"}
var nativePresetPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)
var presetIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var mapPresetMutex sync.Mutex

type MapGeneratorDefaults struct {
	MapGenSettings json.RawMessage `json:"map_gen_settings"`
	MapSettings    json.RawMessage `json:"map_settings"`
	NativePresets  []string        `json:"native_presets"`
}

type WorldCreationRequest struct {
	Name           string                 `json:"name"`
	Preset         string                 `json:"preset"`
	Seed           *uint32                `json:"seed,omitempty"`
	MapGenSettings map[string]interface{} `json:"map_gen_settings"`
	MapSettings    map[string]interface{} `json:"map_settings"`
}

type MapPreset struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	MapGenSettings map[string]interface{} `json:"map_gen_settings"`
	MapSettings    map[string]interface{} `json:"map_settings"`
}

func readJSONObject(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]interface{}
	if err = json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("invalid Factorio example %s: %w", path, err)
	}
	return data, nil
}

func LoadMapGeneratorDefaults() (MapGeneratorDefaults, error) {
	config := bootstrap.GetConfig()
	mapGen, err := readJSONObject(filepath.Join(config.FactorioDir, "data", "map-gen-settings.example.json"))
	if err != nil {
		return MapGeneratorDefaults{}, err
	}
	mapSettings, err := readJSONObject(filepath.Join(config.FactorioDir, "data", "map-settings.example.json"))
	if err != nil {
		return MapGeneratorDefaults{}, err
	}
	return MapGeneratorDefaults{MapGenSettings: mapGen, MapSettings: mapSettings, NativePresets: append([]string{}, nativeMapPresets...)}, nil
}

func validateJSONObject(value map[string]interface{}, label string) error {
	if value == nil {
		return fmt.Errorf("%s must be a JSON object", label)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", label, err)
	}
	if len(encoded) > 1024*1024 {
		return fmt.Errorf("%s exceeds 1 MiB", label)
	}
	return nil
}

func validateNativePreset(value string) error {
	if value == "" {
		return nil
	}
	if !nativePresetPattern.MatchString(value) {
		return fmt.Errorf("invalid native preset name")
	}
	// Mods may add presets. The Factorio process remains the authority that
	// validates whether this syntactically safe preset exists.
	return nil
}

func writeTemporaryJSON(value map[string]interface{}, pattern string) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(value); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func validateCreatedSave(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return errors.New("Factorio created an empty save")
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("Factorio output is not a valid save archive: %w", err)
	}
	defer archive.Close()
	if len(archive.File) == 0 {
		return errors.New("Factorio created an empty save archive")
	}
	return nil
}

func factorioCommand(config bootstrap.Config, args []string) *exec.Cmd {
	return factorioCommandContext(context.Background(), config, args)
}

func factorioCommandContext(ctx context.Context, config bootstrap.Config, args []string) *exec.Cmd {
	if config.GlibcCustom == "true" {
		loaderArgs := []string{"--library-path", config.GlibcLibLoc, config.FactorioBinary, "--executable-path", config.FactorioBinary}
		return exec.CommandContext(ctx, config.GlibcLocation, append(loaderArgs, args...)...)
	}
	return exec.CommandContext(ctx, config.FactorioBinary, args...)
}

func mapConfigurationArgs(request WorldCreationRequest) ([]string, func(), error) {
	if err := validateNativePreset(request.Preset); err != nil {
		return nil, func() {}, err
	}
	if err := validateJSONObject(request.MapGenSettings, "map-gen-settings"); err != nil {
		return nil, func() {}, err
	}
	if err := validateJSONObject(request.MapSettings, "map-settings"); err != nil {
		return nil, func() {}, err
	}
	mapGenPath, err := writeTemporaryJSON(request.MapGenSettings, "fsm-map-gen-*.json")
	if err != nil {
		return nil, func() {}, err
	}
	mapSettingsPath, err := writeTemporaryJSON(request.MapSettings, "fsm-map-settings-*.json")
	if err != nil {
		_ = os.Remove(mapGenPath)
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.Remove(mapGenPath); _ = os.Remove(mapSettingsPath) }
	args := []string{"--map-gen-settings", mapGenPath, "--map-settings", mapSettingsPath}
	if request.Preset != "" {
		args = append(args, "--preset", request.Preset)
	}
	if request.Seed != nil {
		args = append(args, "--map-gen-seed", strconv.FormatUint(uint64(*request.Seed), 10))
	}
	return args, cleanup, nil
}

func CreateWorld(request WorldCreationRequest) (Save, string, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	name, err := NormalizeSaveName(request.Name)
	if err != nil {
		return Save{}, "", err
	}
	config := bootstrap.GetConfig()
	target, err := ResolveDataPath(config.FactorioSavesDir, name)
	if err != nil {
		return Save{}, "", err
	}
	if _, err = os.Stat(target); err == nil {
		return Save{}, "", fmt.Errorf("save %q already exists", name)
	} else if !os.IsNotExist(err) {
		return Save{}, "", err
	}
	mapArgs, cleanup, err := mapConfigurationArgs(request)
	if err != nil {
		return Save{}, "", err
	}
	defer cleanup()
	args := append([]string{"--create", target}, mapArgs...)
	if err = os.MkdirAll(config.FactorioSavesDir, 0755); err != nil {
		return Save{}, "", err
	}
	command := factorioCommand(config, args)
	output, commandErr := command.CombinedOutput()
	if commandErr != nil {
		_ = os.Remove(target)
		return Save{}, string(output), fmt.Errorf("Factorio rejected the world configuration: %w", commandErr)
	}
	if err = validateCreatedSave(target); err != nil {
		_ = os.Remove(target)
		return Save{}, string(output), err
	}
	info, err := os.Stat(target)
	if err != nil {
		return Save{}, string(output), err
	}
	return Save{Name: name, LastMod: info.ModTime(), Size: info.Size()}, string(output), nil
}

func GenerateMapPreview(request WorldCreationRequest) ([]byte, string, error) {
	return GenerateMapPreviewContext(context.Background(), request)
}

// GenerateMapPreviewContext runs a preview in the request's lifetime. When a
// newer browser request cancels the old one, CommandContext terminates the
// obsolete Factorio process instead of leaving previews queued behind it.
func GenerateMapPreviewContext(ctx context.Context, request WorldCreationRequest) ([]byte, string, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	mapArgs, cleanup, err := mapConfigurationArgs(request)
	if err != nil {
		return nil, "", err
	}
	defer cleanup()
	preview, err := os.CreateTemp("", "fsm-map-preview-*.png")
	if err != nil {
		return nil, "", err
	}
	previewPath := preview.Name()
	_ = preview.Close()
	_ = os.Remove(previewPath)
	defer os.Remove(previewPath)
	args := append([]string{"--generate-map-preview", previewPath, "--map-preview-size", "768"}, mapArgs...)
	output, commandErr := factorioCommandContext(ctx, bootstrap.GetConfig(), args).CombinedOutput()
	if commandErr != nil {
		return nil, string(output), fmt.Errorf("Factorio could not generate the preview: %w", commandErr)
	}
	data, err := os.ReadFile(previewPath)
	if err != nil || len(data) == 0 {
		if err == nil {
			err = errors.New("Factorio generated an empty preview")
		}
		return nil, string(output), err
	}
	return data, string(output), nil
}

func validatePresetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return fmt.Errorf("preset name must contain between 1 and 80 characters")
	}
	for _, character := range name {
		if character == 0 || unicode.IsControl(character) {
			return fmt.Errorf("preset name contains invalid characters")
		}
	}
	return nil
}

func presetID(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(sum[:])
}

func SaveMapPreset(preset MapPreset) (MapPreset, error) {
	mapPresetMutex.Lock()
	defer mapPresetMutex.Unlock()
	if err := validatePresetName(preset.Name); err != nil {
		return MapPreset{}, err
	}
	if err := validateJSONObject(preset.MapGenSettings, "map-gen-settings"); err != nil {
		return MapPreset{}, err
	}
	if err := validateJSONObject(preset.MapSettings, "map-settings"); err != nil {
		return MapPreset{}, err
	}
	config := bootstrap.GetConfig()
	if err := os.MkdirAll(config.MapPresetDir, 0755); err != nil {
		return MapPreset{}, err
	}
	preset.ID = presetID(preset.Name)
	path, err := ResolveDataPath(config.MapPresetDir, preset.ID+".json")
	if err != nil {
		return MapPreset{}, err
	}
	now := time.Now().UTC()
	preset.CreatedAt = now
	if data, readErr := os.ReadFile(path); readErr == nil {
		var existing MapPreset
		if json.Unmarshal(data, &existing) == nil && !existing.CreatedAt.IsZero() {
			preset.CreatedAt = existing.CreatedAt
		}
	}
	preset.UpdatedAt = now
	if err = writeJSONFile(path, preset); err != nil {
		return MapPreset{}, err
	}
	return preset, nil
}

func ListMapPresets() ([]MapPreset, error) {
	mapPresetMutex.Lock()
	defer mapPresetMutex.Unlock()
	config := bootstrap.GetConfig()
	entries, err := os.ReadDir(config.MapPresetDir)
	if os.IsNotExist(err) {
		return []MapPreset{}, nil
	}
	if err != nil {
		return nil, err
	}
	presets := make([]MapPreset, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(config.MapPresetDir, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		var preset MapPreset
		if json.Unmarshal(data, &preset) != nil || preset.ID+".json" != entry.Name() {
			continue
		}
		presets = append(presets, preset)
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].Name < presets[j].Name })
	return presets, nil
}

func RemoveMapPreset(id string) error {
	mapPresetMutex.Lock()
	defer mapPresetMutex.Unlock()
	if !presetIDPattern.MatchString(id) {
		return fmt.Errorf("invalid preset id")
	}
	path, err := ResolveDataPath(bootstrap.GetConfig().MapPresetDir, id+".json")
	if err != nil {
		return err
	}
	return os.Remove(path)
}

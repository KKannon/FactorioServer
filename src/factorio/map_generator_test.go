package factorio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

func TestNativePresetValidation(t *testing.T) {
	for _, preset := range nativeMapPresets {
		if err := validateNativePreset(preset); err != nil {
			t.Errorf("native preset %q rejected: %v", preset, err)
		}
	}
	if err := validateNativePreset("mod-added-preset"); err != nil {
		t.Errorf("safe mod-added preset rejected: %v", err)
	}
	for _, preset := range []string{"../default", "death world", "UPPERCASE"} {
		if err := validateNativePreset(preset); err == nil {
			t.Errorf("unsafe or unknown preset %q accepted", preset)
		}
	}
}

func TestMapPresetIDIsStableAndCaseInsensitive(t *testing.T) {
	if presetID(" My Preset ") != presetID("my preset") {
		t.Fatal("equivalent preset names generated different ids")
	}
	if !presetIDPattern.MatchString(presetID("My Preset")) {
		t.Fatal("generated preset id is not a safe filename component")
	}
}

func TestMapSettingsSizeLimit(t *testing.T) {
	if err := validateJSONObject(map[string]interface{}{"width": 0}, "map-gen-settings"); err != nil {
		t.Fatal(err)
	}
	if err := validateJSONObject(map[string]interface{}{"large": strings.Repeat("x", 1024*1024)}, "map-gen-settings"); err == nil {
		t.Fatal("oversized map settings were accepted")
	}
}

func TestFactorioCommandContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := factorioCommandContext(ctx, bootstrap.Config{FactorioBinary: os.Args[0]}, []string{"-test.run=^$"})
	if err := command.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled command returned %v, want context.Canceled", err)
	}
}

func TestCreatedSaveMustBeNonEmptyZip(t *testing.T) {
	root := t.TempDir()
	invalid := filepath.Join(root, "invalid.zip")
	if err := os.WriteFile(invalid, []byte("not a save"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateCreatedSave(invalid); err == nil {
		t.Fatal("invalid archive was accepted")
	}
	valid := filepath.Join(root, "valid.zip")
	writeTestSave(t, valid, "world")
	if err := validateCreatedSave(valid); err != nil {
		t.Fatalf("valid archive rejected: %v", err)
	}
}

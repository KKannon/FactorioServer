package factorio

import (
	"path/filepath"
	"testing"
)

func TestValidateFileName(t *testing.T) {
	for _, value := range []string{"world.zip", "Meu Mundo.zip", "mod_1.0.0.zip"} {
		if err := ValidateFileName(value); err != nil {
			t.Errorf("ValidateFileName(%q) = %v", value, err)
		}
	}
	for _, value := range []string{"", ".", "..", "../world.zip", `folder\world.zip`, "bad\x00name.zip"} {
		if err := ValidateFileName(value); err == nil {
			t.Errorf("ValidateFileName(%q) accepted an unsafe value", value)
		}
	}
}

func TestNormalizeSaveName(t *testing.T) {
	for input, expected := range map[string]string{"world": "world.zip", "world.zip": "world.zip", " Meu Mundo ": "Meu Mundo.zip"} {
		actual, err := NormalizeSaveName(input)
		if err != nil || actual != expected {
			t.Errorf("NormalizeSaveName(%q) = %q, %v; want %q", input, actual, err, expected)
		}
	}
	for _, value := range []string{"../world", "world.exe", "folder/world.zip"} {
		if _, err := NormalizeSaveName(value); err == nil {
			t.Errorf("NormalizeSaveName(%q) accepted an unsafe value", value)
		}
	}
}

func TestResolveDataPath(t *testing.T) {
	root := t.TempDir()
	path, err := ResolveDataPath(root, "world.zip")
	if err != nil || path != filepath.Join(root, "world.zip") {
		t.Fatalf("ResolveDataPath returned %q, %v", path, err)
	}
	if _, err := ResolveDataPath(root, "../world.zip"); err == nil {
		t.Fatal("ResolveDataPath accepted traversal")
	}
}

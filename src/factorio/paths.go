package factorio

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ValidateFileName accepts a single filesystem component and rejects values
// that could escape a configured Factorio data directory.
func ValidateFileName(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return fmt.Errorf("file name is required")
	}
	if filepath.Base(value) != value || strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("file name must not contain a path")
	}
	for _, character := range value {
		if character == 0 || unicode.IsControl(character) {
			return fmt.Errorf("file name contains invalid characters")
		}
	}
	return nil
}

func ValidateSaveName(value string) error {
	if err := ValidateFileName(value); err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Ext(value), ".zip") {
		return fmt.Errorf("save file must use the .zip extension")
	}
	return nil
}

func NormalizeSaveName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if filepath.Ext(value) == "" {
		value += ".zip"
	}
	if err := ValidateSaveName(value); err != nil {
		return "", err
	}
	return value, nil
}

func ResolveDataPath(root, name string) (string, error) {
	if err := ValidateFileName(name); err != nil {
		return "", err
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetPath, err := filepath.Abs(filepath.Join(rootPath, name))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootPath, targetPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes the configured data directory")
	}
	return targetPath, nil
}

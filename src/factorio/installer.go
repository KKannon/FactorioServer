package factorio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

var (
	installMutex          sync.Mutex
	installVersionPattern = regexp.MustCompile(`^(stable|latest|[0-9]+\.[0-9]+\.[0-9]+)$`)
)

func ValidInstallVersion(value string) bool {
	return installVersionPattern.MatchString(strings.TrimSpace(value))
}

func runInstallCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func swapInstallDirectory(tempDir, extractedRoot, factorioDir, name string) (func(), error) {
	target := filepath.Join(factorioDir, name)
	source := filepath.Join(extractedRoot, name)
	backup := filepath.Join(tempDir, "previous-"+name)
	if _, err := os.Stat(source); err != nil {
		return nil, fmt.Errorf("download does not contain %s: %w", name, err)
	}
	hadPrevious := false
	if _, err := os.Stat(target); err == nil {
		if err = os.Rename(target, backup); err != nil {
			return nil, fmt.Errorf("back up current %s: %w", name, err)
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Rename(source, target); err != nil {
		if hadPrevious {
			_ = os.Rename(backup, target)
		}
		return nil, fmt.Errorf("install new %s: %w", name, err)
	}
	return func() {
		_ = os.RemoveAll(target)
		if hadPrevious {
			_ = os.Rename(backup, target)
		}
	}, nil
}

func persistInstallVersion(config bootstrap.Config, requested string) error {
	path := filepath.Join(config.FactorioConfigDir, "fsm-version")
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(requested+"\n"), 0644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// InstallVersion downloads and activates a headless Factorio release without
// replacing the manager image. The game server must already be stopped.
func InstallVersion(requested string) (Version, error) {
	requested = strings.TrimSpace(requested)
	if !ValidInstallVersion(requested) {
		return NilVersion, fmt.Errorf("version must be stable, latest, or a value such as 2.0.77")
	}
	installMutex.Lock()
	defer installMutex.Unlock()

	server := GetFactorioServer()
	if server.GetRunning() {
		return NilVersion, fmt.Errorf("Factorio server must be stopped before changing version")
	}
	config := bootstrap.GetConfig()
	tempDir, err := os.MkdirTemp(config.FactorioDir, ".fsm-version-")
	if err != nil {
		return NilVersion, err
	}
	defer os.RemoveAll(tempDir)

	archive := filepath.Join(tempDir, "factorio.tar.xz")
	downloadURL := "https://www.factorio.com/get-download/" + requested + "/headless/linux64"
	if err = runInstallCommand("curl", "--fail", "--location", "--silent", "--show-error", downloadURL, "--output", archive); err != nil {
		return NilVersion, fmt.Errorf("download Factorio %s: %w", requested, err)
	}
	if err = runInstallCommand("tar", "-xf", archive, "-C", tempDir); err != nil {
		return NilVersion, fmt.Errorf("extract Factorio %s: %w", requested, err)
	}
	extractedRoot := filepath.Join(tempDir, "factorio")
	rollbackBin, err := swapInstallDirectory(tempDir, extractedRoot, config.FactorioDir, "bin")
	if err != nil {
		return NilVersion, err
	}
	rollbackData, err := swapInstallDirectory(tempDir, extractedRoot, config.FactorioDir, "data")
	if err != nil {
		rollbackBin()
		return NilVersion, err
	}
	rollback := func() {
		rollbackData()
		rollbackBin()
		_ = RefreshInstalledVersion()
	}
	if err = RefreshInstalledVersion(); err != nil {
		rollback()
		return NilVersion, fmt.Errorf("validate installed Factorio: %w", err)
	}
	if err = persistInstallVersion(config, requested); err != nil {
		rollback()
		return NilVersion, fmt.Errorf("persist selected version: %w", err)
	}
	return GetFactorioServer().Version, nil
}

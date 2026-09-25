package factorio

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

const backupMarker = "--fsm-backup--"

var saveFileMutex sync.Mutex

type Backup struct {
	Name      string    `json:"name"`
	SaveName  string    `json:"save_name"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"`
}

func backupName(saveName string, createdAt time.Time) (string, error) {
	saveName, err := NormalizeSaveName(saveName)
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(saveName, filepath.Ext(saveName))
	return base + backupMarker + createdAt.UTC().Format("20060102T150405.000000000Z") + ".zip", nil
}

func parseBackupName(name string) (Backup, error) {
	if err := ValidateSaveName(name); err != nil {
		return Backup{}, err
	}
	withoutExtension := strings.TrimSuffix(name, filepath.Ext(name))
	marker := strings.LastIndex(withoutExtension, backupMarker)
	if marker <= 0 {
		return Backup{}, fmt.Errorf("invalid backup name")
	}
	createdAt, err := time.Parse("20060102T150405.000000000Z", withoutExtension[marker+len(backupMarker):])
	if err != nil {
		return Backup{}, fmt.Errorf("invalid backup timestamp: %w", err)
	}
	saveName := withoutExtension[:marker] + ".zip"
	if err := ValidateSaveName(saveName); err != nil {
		return Backup{}, err
	}
	return Backup{Name: name, SaveName: saveName, CreatedAt: createdAt}, nil
}

func copyFile(sourcePath, destinationPath string) (written int64, resultErr error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0664)
	if err != nil {
		return 0, err
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.Remove(destinationPath)
		}
	}()
	written, copyErr := io.Copy(destination, source)
	syncErr := destination.Sync()
	closeErr := destination.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if syncErr != nil {
		return written, syncErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	completed = true
	return written, nil
}

func createBackupAt(savesDir, backupsDir, saveName string, createdAt time.Time) (Backup, error) {
	saveName, err := NormalizeSaveName(saveName)
	if err != nil {
		return Backup{}, err
	}
	sourcePath, err := ResolveDataPath(savesDir, saveName)
	if err != nil {
		return Backup{}, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("save is not a file")
		}
		return Backup{}, err
	}
	if err = os.MkdirAll(backupsDir, 0755); err != nil {
		return Backup{}, err
	}
	name, err := backupName(saveName, createdAt)
	if err != nil {
		return Backup{}, err
	}
	destinationPath, err := ResolveDataPath(backupsDir, name)
	if err != nil {
		return Backup{}, err
	}
	size, err := copyFile(sourcePath, destinationPath)
	if err != nil {
		return Backup{}, err
	}
	return Backup{Name: name, SaveName: saveName, CreatedAt: createdAt.UTC(), Size: size}, nil
}

func CreateSaveBackup(saveName string) (Backup, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	config := bootstrap.GetConfig()
	return createBackupAt(config.FactorioSavesDir, config.FactorioBackupDir, saveName, time.Now())
}

func RemoveSaveWithBackup(saveName string) (Save, Backup, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	save, err := FindSave(saveName)
	if err != nil {
		return Save{}, Backup{}, err
	}
	config := bootstrap.GetConfig()
	backup, err := createBackupAt(config.FactorioSavesDir, config.FactorioBackupDir, save.Name, time.Now())
	if err != nil {
		return Save{}, Backup{}, err
	}
	if err = save.Remove(); err != nil {
		return Save{}, backup, err
	}
	server := GetFactorioServer()
	server.mu.Lock()
	selectionChanged := server.Savefile == save.Name
	if selectionChanged {
		server.Savefile = ""
	}
	server.mu.Unlock()
	if selectionChanged {
		server.broadcastStatus()
	}
	return *save, backup, nil
}

func ListSaveBackups() ([]Backup, error) {
	config := bootstrap.GetConfig()
	entries, err := os.ReadDir(config.FactorioBackupDir)
	if os.IsNotExist(err) {
		return []Backup{}, nil
	}
	if err != nil {
		return nil, err
	}
	backups := make([]Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		backup, parseErr := parseBackupName(entry.Name())
		if parseErr != nil {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		backup.Size = info.Size()
		backups = append(backups, backup)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt.After(backups[j].CreatedAt) })
	return backups, nil
}

func findSaveBackupAt(backupsDir, name string) (Backup, string, error) {
	backup, err := parseBackupName(name)
	if err != nil {
		return Backup{}, "", err
	}
	path, err := ResolveDataPath(backupsDir, backup.Name)
	if err != nil {
		return Backup{}, "", err
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("backup is not a file")
		}
		return Backup{}, "", err
	}
	backup.Size = info.Size()
	return backup, path, nil
}

func FindSaveBackup(name string) (Backup, string, error) {
	return findSaveBackupAt(bootstrap.GetConfig().FactorioBackupDir, name)
}

func restoreSaveBackupAt(savesDir, backupsDir, name string, restoredAt time.Time) (Backup, *Backup, error) {
	backup, sourcePath, err := findSaveBackupAt(backupsDir, name)
	if err != nil {
		return Backup{}, nil, err
	}
	archive, err := zip.OpenReader(sourcePath)
	if err != nil {
		return Backup{}, nil, fmt.Errorf("backup is not a valid ZIP archive: %w", err)
	}
	if len(archive.File) == 0 {
		_ = archive.Close()
		return Backup{}, nil, fmt.Errorf("backup ZIP is empty")
	}
	if err = archive.Close(); err != nil {
		return Backup{}, nil, err
	}
	targetPath, err := ResolveDataPath(savesDir, backup.SaveName)
	if err != nil {
		return Backup{}, nil, err
	}
	var safetyBackup *Backup
	if _, statErr := os.Stat(targetPath); statErr == nil {
		created, backupErr := createBackupAt(savesDir, backupsDir, backup.SaveName, restoredAt)
		if backupErr != nil {
			return Backup{}, nil, fmt.Errorf("create safety backup: %w", backupErr)
		}
		safetyBackup = &created
	} else if !os.IsNotExist(statErr) {
		return Backup{}, nil, statErr
	}
	if err = os.MkdirAll(savesDir, 0755); err != nil {
		return Backup{}, safetyBackup, err
	}
	temporary, err := os.CreateTemp(savesDir, ".fsm-restore-*.zip")
	if err != nil {
		return Backup{}, safetyBackup, err
	}
	temporaryPath := temporary.Name()
	if err = temporary.Close(); err != nil {
		return Backup{}, safetyBackup, err
	}
	_ = os.Remove(temporaryPath)
	defer os.Remove(temporaryPath)
	if _, err = copyFile(sourcePath, temporaryPath); err != nil {
		return Backup{}, safetyBackup, err
	}
	var previousPath string
	if _, statErr := os.Stat(targetPath); statErr == nil {
		holder, createErr := os.CreateTemp(savesDir, ".fsm-previous-*.zip")
		if createErr != nil {
			return Backup{}, safetyBackup, createErr
		}
		previousPath = holder.Name()
		_ = holder.Close()
		_ = os.Remove(previousPath)
		if err = os.Rename(targetPath, previousPath); err != nil {
			return Backup{}, safetyBackup, err
		}
	}
	if err = os.Rename(temporaryPath, targetPath); err != nil {
		if previousPath != "" {
			_ = os.Rename(previousPath, targetPath)
		}
		return Backup{}, safetyBackup, err
	}
	if previousPath != "" {
		_ = os.Remove(previousPath)
	}
	return backup, safetyBackup, nil
}

func RestoreSaveBackup(name string) (Backup, *Backup, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	config := bootstrap.GetConfig()
	return restoreSaveBackupAt(config.FactorioSavesDir, config.FactorioBackupDir, name, time.Now())
}

func RemoveSaveBackup(name string) error {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	_, path, err := findSaveBackupAt(bootstrap.GetConfig().FactorioBackupDir, name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

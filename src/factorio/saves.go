package factorio

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type Save struct {
	Name    string    `json:"name"`
	LastMod time.Time `json:"last_mod"`
	Size    int64     `json:"size"`
	Active  bool      `json:"active"`
}

func (s *Save) String() string {
	return s.Name
}

// Lists save files in factorio/saves
func ListSaves() (saves []Save, err error) {
	config := bootstrap.GetConfig()
	saves = []Save{}
	entries, err := os.ReadDir(config.FactorioSavesDir)
	if err != nil {
		return saves, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".zip") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return saves, infoErr
		}
		saves = append(saves, Save{
			Name:    info.Name(),
			LastMod: info.ModTime(),
			Size:    info.Size(),
		})
	}
	selected := GetFactorioServer().Status().Savefile
	if strings.HasPrefix(selected, "Load Latest") {
		var latestIndex = -1
		for index := range saves {
			if latestIndex == -1 || saves[latestIndex].LastMod.Before(saves[index].LastMod) {
				latestIndex = index
			}
		}
		if latestIndex >= 0 {
			saves[latestIndex].Active = true
		}
	} else {
		for index := range saves {
			saves[index].Active = saves[index].Name == selected
		}
	}
	return
}

func RenameSave(currentName, requestedName string) (Save, error) {
	saveFileMutex.Lock()
	defer saveFileMutex.Unlock()
	currentName, err := NormalizeSaveName(currentName)
	if err != nil {
		return Save{}, err
	}
	requestedName, err = NormalizeSaveName(requestedName)
	if err != nil {
		return Save{}, err
	}
	config := bootstrap.GetConfig()
	currentPath, err := ResolveDataPath(config.FactorioSavesDir, currentName)
	if err != nil {
		return Save{}, err
	}
	requestedPath, err := ResolveDataPath(config.FactorioSavesDir, requestedName)
	if err != nil {
		return Save{}, err
	}
	if _, err = os.Stat(currentPath); err != nil {
		return Save{}, err
	}
	if _, err = os.Stat(requestedPath); err == nil {
		return Save{}, fmt.Errorf("save %q already exists", requestedName)
	} else if !os.IsNotExist(err) {
		return Save{}, err
	}
	if err = os.Rename(currentPath, requestedPath); err != nil {
		return Save{}, err
	}
	server := GetFactorioServer()
	server.mu.Lock()
	changed := false
	if server.Savefile == currentName {
		server.Savefile = requestedName
		changed = true
	}
	server.mu.Unlock()
	if changed {
		server.broadcastStatus()
	}
	info, err := os.Stat(requestedPath)
	if err != nil {
		return Save{}, err
	}
	return Save{Name: requestedName, LastMod: info.ModTime(), Size: info.Size(), Active: changed}, nil
}

func FindSave(name string) (*Save, error) {
	if err := ValidateSaveName(name); err != nil {
		return nil, err
	}
	saves, err := ListSaves()
	if err != nil {
		return nil, fmt.Errorf("error listing saves: %v", err)
	}

	for _, save := range saves {
		if save.Name == name {
			return &save, nil
		}
	}

	return nil, errors.New("save not found")
}

func (s *Save) Remove() error {
	config := bootstrap.GetConfig()
	path, err := ResolveDataPath(config.FactorioSavesDir, s.Name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// Create savefiles for Factorio
func CreateSave(filePath string) (string, error) {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		log.Printf("Error in creating Factorio save: %s", err)
		return "", err
	}

	args := []string{"--create", filePath}
	config := bootstrap.GetConfig()
	cmdOutput, err := exec.Command(config.FactorioBinary, args...).Output()
	if err != nil {
		log.Printf("Error in creating Factorio save: %s", err)
		log.Println(string(cmdOutput))
		return "", err
	}

	result := string(cmdOutput)

	return result, nil
}

func GetLatestSave() (save Save, err error) {
	saves, err := ListSaves()
	if err != nil {
		return save, err
	}
	for _, candidate := range saves {
		if save.LastMod.Before(candidate.LastMod) {
			save = candidate
		}
	}
	return
}

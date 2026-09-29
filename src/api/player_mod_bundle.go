package api

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

const playerModBundleFilename = "factorio-server-mods.zip"

type playerModBundleStatus struct {
	Required    bool   `json:"required"`
	Count       int    `json:"count"`
	Fingerprint string `json:"fingerprint"`
	NoticeKey   string `json:"notice_key,omitempty"`
	Filename    string `json:"filename"`
}

type playerModBundlePlan struct {
	Status playerModBundleStatus
	Files  []string
}

func preparePlayerModBundle(modsDir string) (playerModBundlePlan, error) {
	plan := playerModBundlePlan{Status: playerModBundleStatus{Filename: playerModBundleFilename}}
	installed, err := factorio.NewMods(modsDir)
	if err != nil {
		return plan, err
	}

	results := installed.ListInstalledMods().ModsResult
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	hash := sha256.New()
	for _, mod := range results {
		if !mod.Enabled || mod.FileName == "" {
			continue
		}
		path, err := factorio.ResolveDataPath(modsDir, mod.FileName)
		if err != nil {
			return plan, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return plan, err
		}
		plan.Files = append(plan.Files, path)
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00%d\n", mod.FileName, info.Size(), info.ModTime().UnixNano())
	}

	plan.Status.Count = len(plan.Files)
	plan.Status.Required = plan.Status.Count > 0
	plan.Status.Fingerprint = hex.EncodeToString(hash.Sum(nil))[:16]
	return plan, nil
}

func writePlayerModBundle(writer io.Writer, modsDir string, plan playerModBundlePlan) (err error) {
	archive := zip.NewWriter(writer)
	defer func() {
		if closeErr := archive.Close(); err == nil {
			err = closeErr
		}
	}()

	modListPath, err := factorio.ResolveDataPath(modsDir, "mod-list.json")
	if err != nil {
		return err
	}
	if modList, err := os.ReadFile(modListPath); err == nil {
		entry, err := archive.Create("mod-list.json")
		if err != nil {
			return err
		}
		if _, err = entry.Write(modList); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	readme, err := archive.Create("LEIA-ME.txt")
	if err != nil {
		return err
	}
	if _, err = io.WriteString(readme, "Extraia os arquivos ZIP desta pasta para %APPDATA%\\Factorio\\mods e reinicie o Factorio antes de entrar no servidor.\n"); err != nil {
		return err
	}

	for _, path := range plan.Files {
		if err := func() error {
			if err := factorio.FileLock.RLock(path); err != nil {
				return err
			}
			defer factorio.FileLock.RUnlock(path)
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			entry, err := archive.Create(filepath.Base(path))
			if err != nil {
				return err
			}
			_, err = io.Copy(entry, file)
			return err
		}(); err != nil {
			return err
		}
	}
	return nil
}

func GetPlayerModBundleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	plan, err := preparePlayerModBundle(bootstrap.GetConfig().FactorioModsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not inspect enabled server mods: %v", err), http.StatusInternalServerError)
		return
	}
	user, _ := r.Context().Value(authContextKey{}).(AuthUser)
	noticeHash := sha256.Sum256([]byte(user.Subject + ":" + plan.Status.Fingerprint))
	plan.Status.NoticeKey = hex.EncodeToString(noticeHash[:])[:24]
	_ = json.NewEncoder(w).Encode(plan.Status)
}

func DownloadPlayerModBundle(w http.ResponseWriter, _ *http.Request) {
	modsDir := bootstrap.GetConfig().FactorioModsDir
	plan, err := preparePlayerModBundle(modsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not inspect enabled server mods: %v", err), http.StatusInternalServerError)
		return
	}
	if !plan.Status.Required {
		http.Error(w, "This server has no enabled downloadable mods", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+playerModBundleFilename+"\"")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-FSM-Mod-Count", strconv.Itoa(plan.Status.Count))
	if err := writePlayerModBundle(w, modsDir, plan); err != nil {
		// The response may already have started; log-compatible error text remains
		// preferable to silently returning a truncated archive.
		return
	}
}

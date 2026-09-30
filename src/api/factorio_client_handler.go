package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

type factorioClientMod struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

type factorioClientManifest struct {
	SchemaVersion    int                  `json:"schema_version"`
	ServerID         string               `json:"server_id"`
	ServerName       string               `json:"server_name"`
	GameHost         string               `json:"game_host"`
	GamePort         int                  `json:"game_port"`
	IncludeGamePort  bool                 `json:"include_game_port"`
	FactorioVersion  string               `json:"factorio_version"`
	ServerRunning    bool                 `json:"server_running"`
	ServerState      string               `json:"server_state"`
	ModsFingerprint  string               `json:"mods_fingerprint"`
	ModsCount        int                  `json:"mods_count"`
	Mods             []factorioClientMod  `json:"mods"`
	BundleURL        string               `json:"bundle_url"`
	ClientRepository string               `json:"client_repository"`
	LaunchOptions    factorioClientConfig `json:"launch_options"`
}

func factorioClientMods(modsDir string) ([]factorioClientMod, string, error) {
	installed, err := factorio.NewMods(modsDir)
	if err != nil {
		return nil, "", err
	}
	results := installed.ListInstalledMods().ModsResult
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	mods := make([]factorioClientMod, 0)
	fingerprint := sha256.New()
	for _, mod := range results {
		if !mod.Enabled || mod.FileName == "" {
			continue
		}
		path, err := factorio.ResolveDataPath(modsDir, mod.FileName)
		if err != nil {
			return nil, "", err
		}
		if err := factorio.FileLock.RLock(path); err != nil {
			return nil, "", err
		}
		file, err := os.Open(path)
		if err != nil {
			factorio.FileLock.RUnlock(path)
			return nil, "", err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		factorio.FileLock.RUnlock(path)
		if copyErr != nil {
			return nil, "", copyErr
		}
		if closeErr != nil {
			return nil, "", closeErr
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, "", err
		}
		digest := hex.EncodeToString(hash.Sum(nil))
		mods = append(mods, factorioClientMod{Name: mod.Name, Version: fmt.Sprint(mod.Version), Filename: filepath.Base(path), Size: info.Size(), SHA256: digest})
		_, _ = fmt.Fprintf(fingerprint, "%s\x00%s\n", filepath.Base(path), digest)
	}
	return mods, hex.EncodeToString(fingerprint.Sum(nil)), nil
}

func requestHostname(r *http.Request) string {
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = r.Host
	}
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		return hostname
	}
	return strings.Trim(host, "[]")
}

func GetFactorioClientManifest(w http.ResponseWriter, r *http.Request) {
	mods, fingerprint, err := factorioClientMods(bootstrap.GetConfig().FactorioModsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not build Factorio Client manifest: %v", err), http.StatusInternalServerError)
		return
	}
	status := factorio.GetFactorioServer().Status()
	host := requestHostname(r)
	clientConfig, err := loadFactorioClientConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not load Factorio Client configuration: %v", err), http.StatusInternalServerError)
		return
	}
	clientConfig = effectiveFactorioClientConfig(clientConfig, allowsUnauthenticatedPlayers())
	if clientConfig.PublicGameHost != "" {
		host = clientConfig.PublicGameHost
	}
	port := clientConfig.GamePort
	if port == 0 {
		port = status.Port
		if port == 0 {
			port = 34197
		}
	}
	serverTarget := strings.ToLower(host)
	if clientConfig.IncludeGamePort {
		serverTarget += ":" + fmt.Sprint(port)
	}
	serverIDHash := sha256.Sum256([]byte(serverTarget))
	manifest := factorioClientManifest{
		SchemaVersion: 1, ServerID: hex.EncodeToString(serverIDHash[:])[:20], ServerName: host,
		GameHost: host, GamePort: port, IncludeGamePort: clientConfig.IncludeGamePort, FactorioVersion: fmt.Sprint(status.Version), ServerRunning: status.Running, ServerState: status.State,
		ModsFingerprint: fingerprint, ModsCount: len(mods), Mods: mods,
		BundleURL: "/client-api/v1/mods/download", ClientRepository: "https://github.com/Stupid-DLL/Factorio-Client",
		LaunchOptions: clientConfig,
	}
	clientConfigETag, _ := json.Marshal(clientConfig)
	statusETag := sha256.Sum256([]byte(fingerprint + "\x00" + status.State + "\x00" + fmt.Sprint(status.Version) + "\x00" + serverTarget + "\x00" + string(clientConfigETag)))
	etag := `"` + hex.EncodeToString(statusETag[:]) + `"`
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=5")
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_ = json.NewEncoder(w).Encode(manifest)
}

func DownloadFactorioClientModBundle(w http.ResponseWriter, _ *http.Request) {
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=30")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+playerModBundleFilename+"\"")
	if err := writePlayerModBundle(w, modsDir, plan); err != nil {
		return
	}
}

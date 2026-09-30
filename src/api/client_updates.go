package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/gorilla/mux"
)

var factorioClientUpdateFilename = regexp.MustCompile(`^Factorio-Client-(?:Setup-)?[0-9]+\.[0-9]+\.[0-9]+\.exe(?:\.blockmap)?$`)
var factorioClientUpdateVersion = regexp.MustCompile(`(?m)^version:\s*["']?([0-9]+\.[0-9]+\.[0-9]+)["']?\s*$`)

func clientUpdatesDir() string {
	return filepath.Join(filepath.Dir(bootstrap.GetConfig().ConfFile), "client-updates")
}

func validFactorioClientUpdateFilename(name string) bool {
	return name == "latest.yml" || factorioClientUpdateFilename.MatchString(name)
}

func serveFactorioClientUpdate(w http.ResponseWriter, r *http.Request, name string, download bool) {
	if filepath.Base(name) != name || !validFactorioClientUpdateFilename(name) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(clientUpdatesDir(), name)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	if name == "latest.yml" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if download || strings.HasSuffix(name, ".exe") {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	}
	http.ServeFile(w, r, path)
}

func DownloadLatestFactorioClient(w http.ResponseWriter, r *http.Request) {
	contents, err := os.ReadFile(filepath.Join(clientUpdatesDir(), "latest.yml"))
	if err != nil {
		http.Error(w, "Factorio Client update is not available", http.StatusNotFound)
		return
	}
	match := factorioClientUpdateVersion.FindSubmatch(contents)
	if len(match) != 2 {
		http.Error(w, "Factorio Client update manifest is invalid", http.StatusInternalServerError)
		return
	}
	serveFactorioClientUpdate(w, r, "Factorio-Client-Setup-"+string(match[1])+".exe", true)
}

func GetFactorioClientUpdate(w http.ResponseWriter, r *http.Request) {
	serveFactorioClientUpdate(w, r, mux.Vars(r)["filename"], false)
}

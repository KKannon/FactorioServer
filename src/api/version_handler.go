package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

type installVersionRequest struct {
	Version string `json:"version"`
}

type installVersionResponse struct {
	Requested        string `json:"requested"`
	InstalledVersion string `json:"installed_version"`
}

func ListFactorioVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	catalog, err := factorio.AvailableFactorioVersions(r.Context())
	if err != nil {
		log.Printf("Could not load Factorio release catalog: %v", err)
		http.Error(w, "Could not load the Factorio version list", http.StatusBadGateway)
		return
	}
	WriteResponse(w, catalog)
}

func InstallFactorioVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request installVersionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid version request", http.StatusBadRequest)
		return
	}
	request.Version = strings.TrimSpace(request.Version)
	if !factorio.ValidInstallVersion(request.Version) {
		http.Error(w, "Version must be stable, latest, or a semantic version such as 2.0.77", http.StatusBadRequest)
		return
	}
	installed, err := factorio.InstallVersion(request.Version)
	if err != nil {
		log.Printf("Factorio version installation failed: %v", err)
		http.Error(w, "Could not install the requested Factorio version", http.StatusInternalServerError)
		return
	}
	log.Printf("Installed Factorio version %s from selection %s", installed.String(), request.Version)
	WriteResponse(w, installVersionResponse{Requested: request.Version, InstalledVersion: installed.String()})
}

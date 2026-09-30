package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func writeAccessError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if strings.Contains(err.Error(), "RCON") {
		status = http.StatusServiceUnavailable
	}
	http.Error(w, err.Error(), status)
}

type playerAccessRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

func writeAccessOverview(w http.ResponseWriter) {
	overview, err := factorio.ListAccessOverview()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not read Factorio access lists: %v", err), http.StatusInternalServerError)
		return
	}
	WriteResponse(w, overview)
}

func GetPlayerAccess(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	writeAccessOverview(w)
}

func AddWhitelistPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request playerAccessRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	if err := factorio.AddWhitelistPlayer(request.Username); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func RemoveWhitelistPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := factorio.RemoveWhitelistPlayer(mux.Vars(r)["username"]); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func AddAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request playerAccessRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	if err := factorio.AddAdmin(request.Username); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func RemoveAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := factorio.RemoveAdmin(mux.Vars(r)["username"]); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func AddBan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request playerAccessRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	if err := factorio.AddBan(request.Username, request.Reason); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func RemoveBan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := factorio.RemoveBan(mux.Vars(r)["username"]); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

func UpdateWhitelistPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request struct {
		Enabled bool `json:"enabled"`
	}
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	if err := factorio.SetWhitelistEnabled(request.Enabled); err != nil {
		writeAccessError(w, err)
		return
	}
	writeAccessOverview(w)
}

type playerIntelligenceResponse struct {
	Available     bool                          `json:"available"`
	Reason        string                        `json:"reason,omitempty"`
	Error         string                        `json:"error,omitempty"`
	Scope         string                        `json:"scope"`
	Bridge        factorio.PlayerBridgeStatus   `json:"bridge"`
	GeneratedTick int64                         `json:"generated_tick,omitempty"`
	Source        string                        `json:"source,omitempty"`
	Players       []factorio.PlayerIntelligence `json:"players"`
	Capabilities  factorio.PlayerCapabilities   `json:"capabilities"`
}

func GetPlayerIntelligence(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	user, _ := r.Context().Value(authContextKey{}).(AuthUser)
	response := playerIntelligenceResponse{
		Scope: "self", Bridge: factorio.GetPlayerBridgeStatus(), Players: make([]factorio.PlayerIntelligence, 0),
	}
	if user.CanManage {
		response.Scope = "all"
	}
	if !response.Bridge.Installed {
		response.Reason = "bridge_not_installed"
		WriteResponse(w, response)
		return
	}
	if !response.Bridge.Enabled {
		response.Reason = "bridge_disabled"
		WriteResponse(w, response)
		return
	}
	snapshot, err := factorio.ReadPlayerIntelligence()
	if err != nil {
		log.Printf("Could not read player intelligence: %v", err)
		response.Reason = "server_unavailable"
		if user.CanManage {
			response.Error = err.Error()
		}
		WriteResponse(w, response)
		return
	}
	response.Available = true
	response.GeneratedTick = snapshot.GeneratedTick
	response.Source = snapshot.Source
	response.Capabilities = snapshot.Capabilities
	response.Players = factorio.VisiblePlayerIntelligence(snapshot, user.FactorioUsername(), user.CanManage)
	WriteResponse(w, response)
}

func InstallPlayerBridge(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := factorio.InstallPlayerBridge(); err != nil {
		http.Error(w, fmt.Sprintf("Could not install the player intelligence bridge: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	WriteResponse(w, factorio.GetPlayerBridgeStatus())
}

func DownloadPlayerBridge(w http.ResponseWriter, _ *http.Request) {
	archive, filename, err := factorio.PlayerBridgeArchive()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not build the player intelligence bridge: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive)
}

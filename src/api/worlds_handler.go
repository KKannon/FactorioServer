package api

import (
	"fmt"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

type renameSaveRequest struct {
	Name string `json:"name"`
}

func ListSaveBackups(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	backups, err := factorio.ListSaveBackups()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not list backups: %v", err), http.StatusInternalServerError)
		return
	}
	WriteResponse(w, backups)
}

func CreateSaveBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	backup, err := factorio.CreateSaveBackup(mux.Vars(r)["save"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create backup: %v", err), http.StatusBadRequest)
		return
	}
	WriteResponse(w, backup)
}

func RestoreSaveBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	restored, safetyBackup, err := factorio.RestoreSaveBackup(mux.Vars(r)["backup"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not restore backup: %v", err), http.StatusBadRequest)
		return
	}
	WriteResponse(w, map[string]interface{}{"restored": restored, "safety_backup": safetyBackup})
}

func RemoveSaveBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if err := factorio.RemoveSaveBackup(mux.Vars(r)["backup"]); err != nil {
		http.Error(w, fmt.Sprintf("Could not remove backup: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func DLBackup(w http.ResponseWriter, r *http.Request) {
	backup, path, err := factorio.FindSaveBackup(mux.Vars(r)["backup"])
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", backup.Name))
	http.ServeFile(w, r, path)
}

func RenameSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var request renameSaveRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	renamed, err := factorio.RenameSave(mux.Vars(r)["save"], request.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not rename save: %v", err), http.StatusBadRequest)
		return
	}
	WriteResponse(w, renamed)
}

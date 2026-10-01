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

type bulkNamesRequest struct {
	Names []string `json:"names"`
}
type bulkOperationResult struct {
	Name   string      `json:"name"`
	OK     bool        `json:"ok"`
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

func readBulkNames(w http.ResponseWriter, r *http.Request) ([]string, bool) {
	var request bulkNamesRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return nil, false
	}
	if len(request.Names) == 0 || len(request.Names) > 100 {
		http.Error(w, "select between 1 and 100 items", http.StatusBadRequest)
		return nil, false
	}
	seen := map[string]bool{}
	names := make([]string, 0, len(request.Names))
	for _, name := range request.Names {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names, true
}

func BulkCreateSaveBackups(w http.ResponseWriter, r *http.Request) {
	names, ok := readBulkNames(w, r)
	if !ok {
		return
	}
	results := make([]bulkOperationResult, 0, len(names))
	for _, name := range names {
		backup, err := factorio.CreateSaveBackup(name)
		item := bulkOperationResult{Name: name, OK: err == nil, Result: backup}
		if err != nil {
			item.Error = err.Error()
			item.Result = nil
		}
		results = append(results, item)
	}
	WriteResponse(w, results)
}

func BulkRemoveSaves(w http.ResponseWriter, r *http.Request) {
	names, ok := readBulkNames(w, r)
	if !ok {
		return
	}
	results := make([]bulkOperationResult, 0, len(names))
	for _, name := range names {
		save, backup, err := factorio.RemoveSaveWithBackup(name)
		item := bulkOperationResult{Name: name, OK: err == nil, Result: map[string]interface{}{"save": save, "backup": backup}}
		if err != nil {
			item.Error = err.Error()
			item.Result = nil
		}
		results = append(results, item)
	}
	WriteResponse(w, results)
}

func BulkRestoreSaveBackups(w http.ResponseWriter, r *http.Request) {
	names, ok := readBulkNames(w, r)
	if !ok {
		return
	}
	results := make([]bulkOperationResult, 0, len(names))
	for _, name := range names {
		restored, safety, err := factorio.RestoreSaveBackup(name)
		item := bulkOperationResult{Name: name, OK: err == nil, Result: map[string]interface{}{"restored": restored, "safety_backup": safety}}
		if err != nil {
			item.Error = err.Error()
			item.Result = nil
		}
		results = append(results, item)
	}
	WriteResponse(w, results)
}

func BulkRemoveSaveBackups(w http.ResponseWriter, r *http.Request) {
	names, ok := readBulkNames(w, r)
	if !ok {
		return
	}
	results := make([]bulkOperationResult, 0, len(names))
	for _, name := range names {
		err := factorio.RemoveSaveBackup(name)
		item := bulkOperationResult{Name: name, OK: err == nil}
		if err != nil {
			item.Error = err.Error()
		}
		results = append(results, item)
	}
	WriteResponse(w, results)
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

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type modUploadPolicy struct {
	AllowUnpublishedMods bool `json:"allow_unpublished_mods"`
}

var modUploadPolicyMu sync.Mutex

func modUploadPolicyPath() string {
	return filepath.Join(bootstrap.GetConfig().FactorioConfigDir, "fsm-mod-upload-policy.json")
}

func loadModUploadPolicy() (modUploadPolicy, error) {
	modUploadPolicyMu.Lock()
	defer modUploadPolicyMu.Unlock()
	var policy modUploadPolicy
	data, err := os.ReadFile(modUploadPolicyPath())
	if os.IsNotExist(err) {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	err = json.Unmarshal(data, &policy)
	return policy, err
}

func saveModUploadPolicy(policy modUploadPolicy) error {
	modUploadPolicyMu.Lock()
	defer modUploadPolicyMu.Unlock()
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(modUploadPolicyPath()), 0755); err != nil {
		return err
	}
	temporary := modUploadPolicyPath() + ".tmp"
	if err = os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, modUploadPolicyPath())
}

func GetModUploadPolicy(w http.ResponseWriter, _ *http.Request) {
	policy, err := loadModUploadPolicy()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(policy)
}

func UpdateModUploadPolicy(w http.ResponseWriter, r *http.Request) {
	var policy modUploadPolicy
	if _, err := ReadFromRequestBody(w, r, &policy); err != nil {
		return
	}
	if err := saveModUploadPolicy(policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(policy)
}

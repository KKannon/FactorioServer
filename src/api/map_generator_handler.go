package api

import (
	"fmt"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func MapGeneratorDefaults(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	defaults, err := factorio.LoadMapGeneratorDefaults()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not load Factorio map defaults: %v", err), http.StatusInternalServerError)
		return
	}
	WriteResponse(w, defaults)
}

func CreateWorld(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	var request factorio.WorldCreationRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	save, output, err := factorio.CreateWorld(request)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		WriteResponse(w, map[string]interface{}{"error": err.Error(), "factorio_output": output})
		return
	}
	WriteResponse(w, map[string]interface{}{"save": save, "factorio_output": output})
}

func GenerateMapPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	var request factorio.WorldCreationRequest
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	preview, _, err := factorio.GenerateMapPreviewContext(r.Context(), request)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		http.Error(w, fmt.Sprintf("Could not generate map preview: %v", err), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(preview)
}

func ListMapPresets(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	presets, err := factorio.ListMapPresets()
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not list map presets: %v", err), http.StatusInternalServerError)
		return
	}
	WriteResponse(w, presets)
}

func SaveMapPreset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	var request factorio.MapPreset
	if resp, err := ReadFromRequestBody(w, r, &request); err != nil {
		WriteResponse(w, resp)
		return
	}
	preset, err := factorio.SaveMapPreset(request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not save map preset: %v", err), http.StatusBadRequest)
		return
	}
	WriteResponse(w, preset)
}

func RemoveMapPreset(w http.ResponseWriter, r *http.Request) {
	if err := factorio.RemoveMapPreset(mux.Vars(r)["id"]); err != nil {
		http.Error(w, fmt.Sprintf("Could not delete map preset: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

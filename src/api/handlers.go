package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

const readHttpBodyError = "Could not read the Request Body."

type JSONResponseFileInput struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,string"`
	Error     string      `json:"error"`
	ErrorKeys []int       `json:"errorkeys"`
}

func WriteResponse(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error writing response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ReadRequestBody(w http.ResponseWriter, r *http.Request) (body []byte, resp interface{}, err error) {
	if r.Body == nil {
		resp = fmt.Sprintf("%s: no request body", readHttpBodyError)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		err = errors.New("no request body")
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		resp = fmt.Sprintf("%s: %s", readHttpBodyError, err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
	}
	return
}

// Lists all save files in the factorio/saves directory
func ListSaves(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	latestParam := r.URL.Query().Get("latest")

	var withLatest bool

	if latestParam != "" {
		var err error
		withLatest, err = strconv.ParseBool(latestParam)
		if err != nil {
			resp = fmt.Sprintf("Error parsing latestParam: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	savesList, err := factorio.ListSaves()
	if err != nil {
		resp = fmt.Sprintf("Error listing save files: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get actual latest and add name
	// but only if requested
	if withLatest && len(savesList) != 0 {
		latestSave, err := factorio.GetLatestSave()
		if err != nil {
			resp = fmt.Sprintf("Error getting latest save: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		latestSave.Name = fmt.Sprintf("Load Latest (%s)", latestSave.Name)
		savesList = append(savesList, latestSave)
	}

	resp = savesList
}

func DLSave(w http.ResponseWriter, r *http.Request) {
	config := bootstrap.GetConfig()
	save, err := factorio.FindSave(mux.Vars(r)["save"])
	if err != nil {
		http.Error(w, "Save not found", http.StatusNotFound)
		return
	}
	saveName, err := factorio.ResolveDataPath(config.FactorioSavesDir, save.Name)
	if err != nil {
		http.Error(w, "Invalid save name", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", save.Name))
	log.Printf("%s downloading: %s", r.Host, saveName)

	http.ServeFile(w, r, saveName)
}

func UploadSave(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	log.Println("Uploading save file")

	if err := parseMultipartWithinLimit(w, r); err != nil {
		resp = err.Error()
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	files := uploadedFiles(r, "savefile")
	if len(files) != 1 {
		resp = "Exactly one save file is required"
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := storeUploadedSave(files[0]); err != nil {
		resp = fmt.Sprintf("Could not store save: %s", err)
		if strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}

	resp = "Uploading files successful"
}

// Deletes provided save
func RemoveSave(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	name := vars["save"]

	save, backup, err := factorio.RemoveSaveWithBackup(name)
	if err != nil {
		resp = fmt.Sprintf("Could not safely remove save {%s}: %s", name, err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// save was removed
	resp = fmt.Sprintf("Removed save: %s. Safety backup: %s", save.Name, backup.Name)
}

// Launches Factorio server binary with --create flag to create save
// Url must include save name for creation of savefile
func CreateSaveHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	vars := mux.Vars(r)
	saveName, err := factorio.NormalizeSaveName(vars["save"])
	if err != nil {
		resp = fmt.Sprintf("Invalid save name: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	config := bootstrap.GetConfig()
	saveFile, err := factorio.ResolveDataPath(config.FactorioSavesDir, saveName)
	if err != nil {
		resp = fmt.Sprintf("Invalid save path: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err = os.Stat(saveFile); err == nil {
		resp = fmt.Sprintf("Save %s already exists", saveName)
		w.WriteHeader(http.StatusConflict)
		return
	}
	cmdOut, err := factorio.CreateSave(saveFile)
	if err != nil {
		resp = fmt.Sprintf("Error creating save {%s}: %s", saveName, err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = fmt.Sprintf("Save %s created successfully. Command output: \n%s", saveName, cmdOut)
}

// LogTail returns last lines of the factorio-current.log file
func LogTail(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	config := bootstrap.GetConfig()
	resp, err = factorio.TailLog()
	if err != nil {
		resp = fmt.Sprintf("Could not tail %s: %s", config.FactorioLog, err)
		return
	}
}

// LoadConfig returns JSON response of config.ini file
func LoadConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	config := bootstrap.GetConfig()
	configContents, err := factorio.LoadConfig(config.FactorioConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			resp = map[string]map[string]string{}
			return
		}
		resp = fmt.Sprintf("Could not retrieve config.ini: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = configContents

	log.Printf("Sent config.ini response")
}

func StartServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	var server = factorio.GetFactorioServer()
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	if server.GetRunning() {
		resp = "Factorio server is already running"
		w.WriteHeader(http.StatusConflict)
		return
	}

	body, readResp, err := ReadRequestBody(w, r)
	if err != nil {
		resp = readResp
		return
	}

	var request struct {
		Savefile string `json:"savefile"`
		BindIP   string `json:"bindip"`
		Port     int    `json:"port"`
	}
	err = json.Unmarshal(body, &request)
	if err != nil {
		resp = fmt.Sprintf("Error unmarshalling server settings JSON: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if savefile was submitted with request to start server.
	if request.Savefile == "" {
		resp = "Error starting Factorio server: No save file provided"
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if net.ParseIP(request.BindIP) == nil {
		resp = "Error starting Factorio server: Invalid bind IP"
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if request.Port < 1 || request.Port > 65535 {
		resp = "Error starting Factorio server: Port must be between 1 and 65535"
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(request.Savefile, "Load Latest") {
		if _, err = factorio.FindSave(request.Savefile); err != nil {
			resp = fmt.Sprintf("Error starting Factorio server: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if err = server.PrepareStart(request.Savefile, request.BindIP, request.Port); err != nil {
		resp = err.Error()
		w.WriteHeader(http.StatusConflict)
		return
	}

	go func() {
		if runErr := server.Run(); runErr != nil {
			log.Printf("Error starting Factorio server: %+v", runErr)
			return
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	resp = fmt.Sprintf("Factorio server with save: %s is starting on port: %d", request.Savefile, request.Port)
	log.Println(resp)
}

func StopServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	if server.GetRunning() {
		err := server.Stop()
		if err != nil {
			resp = fmt.Sprintf("Error stopping factorio server: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp = "Factorio server is stopping"
		log.Println(resp)
	} else {
		resp = "Factorio server is not running"
		w.WriteHeader(http.StatusConflict)
		return
	}
}

func RestartServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	server := factorio.GetFactorioServer()
	if server.GetState() != factorio.StateRunning {
		resp = "Factorio server must be running before it can be restarted"
		w.WriteHeader(http.StatusConflict)
		return
	}
	go func() {
		if err := server.Restart(); err != nil {
			log.Printf("Error restarting Factorio server: %v", err)
		}
	}()
	w.WriteHeader(http.StatusAccepted)
	resp = "Factorio server restart requested"
}

func KillServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	if server.GetRunning() {
		err := server.Kill()
		if err != nil {
			resp = fmt.Sprintf("Error killing factorio server: %s", err)
			log.Println(resp)
			return
		}

		log.Printf("Killed Factorio server.")
		resp = fmt.Sprintf("Factorio server killed")
	} else {
		resp = "Factorio server is not running"
		w.WriteHeader(http.StatusBadRequest)
	}
}

func CheckServer(w http.ResponseWriter, r *http.Request) {
	defer func() {
		WriteResponse(w, factorio.GetFactorioServer().Status())
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
}

func FactorioVersion(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	resp["version"] = server.Version.String()
	resp["base_mod_version"] = server.BaseModVersion
}

// GetServerSettings returns JSON response of server-settings.json file
func GetServerSettings(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	resp = server.Settings

	log.Printf("Sent server settings response")
}

func UpdateServerSettings(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}
	var server = factorio.GetFactorioServer()

	// Race Condition while unmarshal possible
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		err = json.Unmarshal(body, &server.Settings)
		wg.Done()
	}()

	// Wait for unmarshal to avoid race condition
	wg.Wait()

	if err != nil {
		resp = fmt.Sprintf("Error unmarhaling server settings JSON: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	settings, err := json.MarshalIndent(&server.Settings, "", "  ")
	if err != nil {
		resp = fmt.Sprintf("Failed to marshal server settings: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	config := bootstrap.GetConfig()
	err = ioutil.WriteFile(config.SettingsFile, settings, 0644)
	if err != nil {
		resp = fmt.Sprintf("Failed to save server settings: %v\n", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("Saved Factorio server settings in server-settings.json")

	if (server.Version.Greater(factorio.Version{0, 17, 0})) {
		// save admins to adminJson
		admins, err := json.MarshalIndent(server.Settings["admins"], "", "  ")
		if err != nil {
			resp = fmt.Sprintf("Failed to marshal admins-Setting: %s", err)
			log.Println(resp)
			return
		}

		err = ioutil.WriteFile(config.FactorioAdminFile, admins, 0664)
		if err != nil {
			resp = fmt.Sprintf("Failed to save admins: %s", err)
			log.Println(resp)
			return
		}
	}

	resp = fmt.Sprintf("Settings successfully saved")
}

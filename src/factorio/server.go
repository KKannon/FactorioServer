package factorio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/rcon"
)

type Server struct {
	mu             sync.RWMutex           `json:"-"`
	rconMu         sync.Mutex             `json:"-"`
	Cmd            *exec.Cmd              `json:"-"`
	Savefile       string                 `json:"savefile"`
	Latency        int                    `json:"latency"`
	BindIP         string                 `json:"bindip"`
	Port           int                    `json:"port"`
	Running        bool                   `json:"running"`
	State          string                 `json:"state"`
	LastError      string                 `json:"last_error,omitempty"`
	RconConnected  bool                   `json:"rcon_connected"`
	Version        Version                `json:"fac_version"`
	BaseModVersion string                 `json:"base_mod_version"`
	StdOut         io.ReadCloser          `json:"-"`
	StdErr         io.ReadCloser          `json:"-"`
	StdIn          io.WriteCloser         `json:"-"`
	Settings       map[string]interface{} `json:"-"`
	Rcon           *rcon.RemoteConsole    `json:"-"`
	LogChan        chan []string          `json:"-"`
	processID      int                    `json:"-"`
	startedAt      time.Time              `json:"-"`
	onlinePlayers  map[string]time.Time   `json:"-"`
	processError   string                 `json:"-"`
}

type ServerStatus struct {
	Savefile       string  `json:"savefile"`
	Latency        int     `json:"latency"`
	BindIP         string  `json:"bindip"`
	Port           int     `json:"port"`
	Running        bool    `json:"running"`
	State          string  `json:"state"`
	LastError      string  `json:"last_error,omitempty"`
	RconConnected  bool    `json:"rcon_connected"`
	Version        Version `json:"fac_version"`
	BaseModVersion string  `json:"base_mod_version"`
}

var instantiated Server
var once sync.Once

const (
	StateStopped  = "stopped"
	StateStarting = "starting"
	StateRunning  = "running"
	StateStopping = "stopping"
	StateError    = "error"
)

func (server *Server) broadcastStatus() {
	response, err := json.Marshal(server.Status())
	if err != nil {
		log.Printf("marshal server status: %v", err)
		return
	}
	websocket.WebsocketHub.GetRoom("server_status").Send(string(response))
}

func (server *Server) Status() ServerStatus {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return ServerStatus{
		Savefile:       server.Savefile,
		Latency:        server.Latency,
		BindIP:         server.BindIP,
		Port:           server.Port,
		Running:        server.Running,
		State:          server.State,
		LastError:      server.LastError,
		RconConnected:  server.RconConnected,
		Version:        server.Version,
		BaseModVersion: server.BaseModVersion,
	}
}

func (server *Server) SetState(state string, lastError string) {
	server.mu.Lock()
	changed := server.State != state || server.LastError != lastError
	server.State = state
	server.LastError = lastError
	server.Running = state == StateStarting || state == StateRunning || state == StateStopping
	if state == StateStopped || state == StateError {
		server.RconConnected = false
		server.processID = 0
		server.startedAt = time.Time{}
		server.onlinePlayers = make(map[string]time.Time)
	}
	server.mu.Unlock()
	if changed {
		server.broadcastStatus()
	}
}

func (server *Server) GetState() string {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.State
}

func (server *Server) SetRconConnected(connected bool) {
	server.mu.Lock()
	changed := server.RconConnected != connected
	server.RconConnected = connected
	server.mu.Unlock()
	if changed {
		server.broadcastStatus()
	}
}

func (server *Server) SetRCON(console *rcon.RemoteConsole) {
	server.rconMu.Lock()
	server.mu.Lock()
	previous := server.Rcon
	changed := server.RconConnected != (console != nil)
	server.Rcon = console
	server.RconConnected = console != nil
	server.mu.Unlock()
	if previous != nil && previous != console {
		if err := previous.Close(); err != nil {
			log.Printf("Error closing previous rcon connection: %s", err)
		}
	}
	server.rconMu.Unlock()
	if changed {
		server.broadcastStatus()
	}
}

func (server *Server) SendRCON(command string) error {
	server.rconMu.Lock()
	defer server.rconMu.Unlock()
	server.mu.RLock()
	console := server.Rcon
	connected := server.RconConnected
	server.mu.RUnlock()
	if console == nil || !connected {
		return errors.New("RCON is not connected")
	}
	_, err := console.Write(command)
	return err
}

func (server *Server) CloseRCON() error {
	server.rconMu.Lock()
	server.mu.Lock()
	console := server.Rcon
	changed := server.RconConnected || console != nil
	server.Rcon = nil
	server.RconConnected = false
	server.mu.Unlock()
	var err error
	if console != nil {
		err = console.Close()
	}
	server.rconMu.Unlock()
	if changed {
		server.broadcastStatus()
	}
	return err
}

func (server *Server) PrepareStart(savefile string, bindIP string, port int) error {
	server.mu.Lock()
	if server.Running {
		server.mu.Unlock()
		return errors.New("Factorio server is already running or changing state")
	}
	server.Savefile = savefile
	server.BindIP = bindIP
	server.Port = port
	server.State = StateStarting
	server.Running = true
	server.RconConnected = false
	server.LastError = ""
	server.processError = ""
	server.mu.Unlock()
	server.broadcastStatus()
	return nil
}

func (server *Server) Restart() error {
	server.mu.RLock()
	savefile := server.Savefile
	bindIP := server.BindIP
	port := server.Port
	running := server.Running
	server.mu.RUnlock()
	if !running {
		return errors.New("Factorio server is not running")
	}
	if err := server.Stop(); err != nil {
		return err
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if server.GetState() == StateStopped {
			if err := server.PrepareStart(savefile, bindIP, port); err != nil {
				return err
			}
			return server.Run()
		}
		time.Sleep(250 * time.Millisecond)
	}
	server.SetState(StateRunning, "restart timed out while waiting for Factorio to stop")
	return errors.New("restart timed out while waiting for Factorio to stop")
}

func (server *Server) SetRunning(newState bool) {
	if newState {
		server.SetState(StateRunning, "")
	} else {
		server.SetState(StateStopped, "")
	}
}

func (server *Server) GetRunning() bool {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.Running
}

func (server *Server) autostart() {
	var err error
	if server.BindIP == "" {
		server.BindIP = "0.0.0.0"

	}
	if server.Port == 0 {
		server.Port = 34197
	}
	server.Savefile = "Load Latest"

	err = server.Run()

	if err != nil {
		log.Printf("Error starting Factorio server: %+v", err)
		return
	}

}

func SetFactorioServer(server Server) {
	if server.State == "" {
		server.State = StateStopped
	}
	instantiated = server
}

func loadInstalledVersion(server *Server, config bootstrap.Config) error {
	var (
		out []byte
		err error
	)
	if config.GlibcCustom == "true" {
		out, err = exec.Command(config.GlibcLocation, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--version").Output()
	} else {
		out, err = exec.Command(config.FactorioBinary, "--version").Output()
	}
	if err != nil {
		return fmt.Errorf("load Factorio version: %w", err)
	}

	match := regexp.MustCompile(`Version.*?((\d+\.)?(\d+\.)?(\*|\d+)+)`).FindStringSubmatch(string(out))
	if len(match) < 2 {
		return fmt.Errorf("Factorio returned an unrecognized version string")
	}
	if err = server.Version.UnmarshalText([]byte(match[1])); err != nil {
		return fmt.Errorf("parse Factorio version: %w", err)
	}

	baseModInfoFile := filepath.Join(config.FactorioBaseModDir, "info.json")
	baseModData, err := ioutil.ReadFile(baseModInfoFile)
	if err != nil {
		return fmt.Errorf("open base mod info: %w", err)
	}
	var modInfo ModInfo
	if err = json.Unmarshal(baseModData, &modInfo); err != nil {
		return fmt.Errorf("parse base mod info: %w", err)
	}
	server.BaseModVersion = modInfo.Version
	return nil
}

// RefreshInstalledVersion reloads the executable and base mod versions after
// an in-place Factorio installation change.
func RefreshInstalledVersion() error {
	return loadInstalledVersion(GetFactorioServer(), bootstrap.GetConfig())
}

func NewFactorioServer() (err error) {
	server := Server{State: StateStopped}
	server.Settings = make(map[string]interface{})
	config := bootstrap.GetConfig()
	if err = os.MkdirAll(config.FactorioConfigDir, 0755); err != nil {
		log.Printf("failed to create config directory: %v", err)
		return
	}

	settingsPath := config.SettingsFile
	var settings *os.File

	if _, err = os.Stat(settingsPath); os.IsNotExist(err) {
		// copy example settings to supplied settings file, if not exists
		log.Printf("Server settings at %s not found, copying example server settings.\n", settingsPath)

		examplePath := filepath.Join(config.FactorioDir, "data", "server-settings.example.json")

		var example *os.File
		example, err = os.Open(examplePath)
		if err != nil {
			log.Printf("failed to open example server settings: %v", err)
			return
		}
		defer example.Close()

		settings, err = os.Create(settingsPath)
		if err != nil {
			log.Printf("failed to create server settings file: %v", err)
			return
		}
		defer settings.Close()

		_, err = io.Copy(settings, example)
		if err != nil {
			log.Printf("failed to copy example server settings: %v", err)
			return
		}

		err = example.Close()
		if err != nil {
			log.Printf("failed to close example server settings: %s", err)
			return
		}
	} else {
		// otherwise, open file normally
		settings, err = os.Open(settingsPath)
		if err != nil {
			log.Printf("failed to open server settings file: %v", err)
			return
		}
		defer settings.Close()
	}

	// before reading reset offset
	if _, err = settings.Seek(0, 0); err != nil {
		log.Printf("error while seeking in settings file: %v", err)
		return
	}

	if err = json.NewDecoder(settings).Decode(&server.Settings); err != nil {
		log.Printf("error reading %s: %v", settingsPath, err)
		return
	}

	log.Printf("Loaded Factorio settings from %s\n", settingsPath)

	if err = loadInstalledVersion(&server, config); err != nil {
		log.Printf("error loading installed Factorio version: %v", err)
		return
	}

	// load admins from additional file
	if (server.Version.Greater(Version{0, 17, 0})) {
		if _, err = os.Stat(config.FactorioAdminFile); os.IsNotExist(err) {
			//save empty admins-file
			err = ioutil.WriteFile(config.FactorioAdminFile, []byte("[]"), 0664)
			server.Settings["admins"] = make([]string, 0)
		} else {
			var data []byte
			data, err = ioutil.ReadFile(config.FactorioAdminFile)
			if err != nil {
				log.Printf("Error loading FactorioAdminFile: %s", err)
				return
			}

			var jsonData interface{}
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				log.Printf("Error unmarshalling FactorioAdminFile: %s", err)
				return
			}

			server.Settings["admins"] = jsonData
		}
	}
	if _, err = os.Stat(config.FactorioWhitelistFile); os.IsNotExist(err) {
		if err = ioutil.WriteFile(config.FactorioWhitelistFile, []byte("[]"), 0664); err != nil {
			return err
		}
	}
	if _, err = os.Stat(config.FactorioBanFile); os.IsNotExist(err) {
		if err = ioutil.WriteFile(config.FactorioBanFile, []byte("[]"), 0664); err != nil {
			return err
		}
	}

	SetFactorioServer(server)

	// autostart factorio is configured to do so
	if config.Autostart == "true" {
		go instantiated.autostart()
	}

	return
}

func GetFactorioServer() (f *Server) {
	return &instantiated
}

func (server *Server) Run() (runErr error) {
	if !server.GetRunning() {
		server.SetState(StateStarting, "")
	}
	defer func() {
		if runErr != nil {
			server.mu.RLock()
			processError := server.processError
			server.mu.RUnlock()
			if processError != "" {
				runErr = errors.New(processError)
			}
			server.SetState(StateError, runErr.Error())
		}
	}()
	var err error
	config := bootstrap.GetConfig()
	data, err := json.MarshalIndent(server.Settings, "", "  ")
	if err != nil {
		log.Println("Failed to marshal FactorioServerSettings: ", err)
	} else {
		ioutil.WriteFile(config.SettingsFile, data, 0644)
	}

	saves, err := ListSaves()
	if err != nil {
		log.Println("Failed to get saves list: ", err)
	}

	if len(saves) == 0 {
		return errors.New("No savefile exists on the server")
	}

	args := []string{}

	//The factorio server refenences its executable-path, since we execute the ld.so file and pass the factorio binary as a parameter
	//the game would use the path to the ld.so file as it's executable path and crash, to prevent this the parameter "--executable-path" is added
	if config.GlibcCustom == "true" {
		log.Println("Custom glibc selected, glibc.so location:", config.GlibcLocation, " lib location:", config.GlibcLibLoc)
		args = append(args, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--executable-path", config.FactorioBinary)
	}

	args = append(args,
		"--bind", server.BindIP,
		"--port", strconv.Itoa(server.Port),
		"--server-settings", config.SettingsFile,
		"--rcon-port", strconv.Itoa(config.FactorioRconPort),
		"--rcon-password", config.FactorioRconPass)

	if (server.Version.Greater(Version{0, 17, 0})) {
		args = append(args, "--server-adminlist", config.FactorioAdminFile)
	}
	args = append(args,
		fmt.Sprintf("--use-server-whitelist=%t", WhitelistEnabled()),
		"--server-whitelist", config.FactorioWhitelistFile,
		"--server-banlist", config.FactorioBanFile)

	if strings.HasPrefix(server.Savefile, "Load Latest") {
		args = append(args, "--start-server-load-latest")
	} else {
		args = append(args, "--start-server", filepath.Join(config.FactorioSavesDir, server.Savefile))
	}

	// Write chat log to a different file if requested (if not it will be mixed-in with the default logfile)
	if config.ChatLogFile != "" {
		args = append(args, "--console-log", config.ChatLogFile)
	}

	if config.GlibcCustom == "true" {
		log.Println("Starting server with command: ", config.GlibcLocation, args)
		server.Cmd = exec.Command(config.GlibcLocation, args...)
	} else {
		log.Println("Starting server with command: ", config.FactorioBinary, args)
		server.Cmd = exec.Command(config.FactorioBinary, args...)
	}

	server.StdOut, err = server.Cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error opening stdout pipe: %s", err)
		return err
	}

	server.StdIn, err = server.Cmd.StdinPipe()
	if err != nil {
		log.Printf("Error opening stdin pipe: %s", err)
		return err
	}

	server.StdErr, err = server.Cmd.StderrPipe()
	if err != nil {
		log.Printf("Error opening stderr pipe: %s", err)
		return err
	}

	go server.parseRunningCommand(server.StdOut)
	go server.parseRunningCommand(server.StdErr)

	err = server.Cmd.Start()
	if err != nil {
		log.Printf("Factorio process failed to start: %s", err)
		return err
	}
	server.mu.Lock()
	server.processID = server.Cmd.Process.Pid
	server.startedAt = time.Now()
	server.onlinePlayers = make(map[string]time.Time)
	server.mu.Unlock()
	server.SetState(StateRunning, "")

	err = server.Cmd.Wait()
	log.Printf("Factorio process is closed")
	wasStopping := server.GetState() == StateStopping
	server.SetState(StateStopped, "")
	if err != nil {
		if wasStopping {
			return nil
		}
		log.Printf("Factorio process exited with error: %s", err)
		return err
	}

	return nil
}

func (server *Server) parseRunningCommand(std io.ReadCloser) (err error) {
	stdScanner := bufio.NewScanner(std)
	for stdScanner.Scan() {
		text := stdScanner.Text()
		server.observePlayerEvent(text)

		log.Printf("Factorio Server: %s", text)
		if err := server.writeLog(text); err != nil {
			log.Printf("Error: %s", err)
		}

		// send the reported line per websocket
		wsRoom := websocket.WebsocketHub.GetRoom("gamelog")
		go wsRoom.Send(text)

		line := strings.Fields(text)
		// Ensure logline slice is in bounds
		if len(line) > 1 {
			// Check if Factorio Server reports any errors if so handle it
			if line[1] == "Error" {
				server.rememberProcessError(text)
				err := server.checkLogError(line)
				if err != nil {
					log.Printf("Error checking Factorio Server Error: %s", err)
				}
			}
			// If rcon port opens indicated in log connect to rcon
			rconLog := "Starting RCON interface at IP"
			// check if slice index is greater than 2 to prevent panic
			if len(line) > 2 {
				// log line for opened rcon connection
				if strings.Contains(text, rconLog) {
					log.Printf("Rcon running on Factorio Server")
					err = connectRC()
					if err != nil {
						log.Printf("Error: %s", err)
					}
				}

				server.checkProcessHealth(text)
			}
		}
	}
	if err := stdScanner.Err(); err != nil {
		log.Printf("Error reading std buffer: %s", err)
		return err
	}
	return nil
}

func (server *Server) rememberProcessError(line string) {
	detail := strings.TrimSpace(line)
	if index := strings.Index(detail, "Error "); index >= 0 {
		detail = strings.TrimSpace(detail[index+len("Error "):])
	}
	if len(detail) > 500 {
		detail = detail[:500]
	}
	server.mu.Lock()
	server.processError = detail
	server.mu.Unlock()
}

func (server *Server) writeLog(logline string) error {
	config := bootstrap.GetConfig()
	logfileName := config.ConsoleLogFile
	file, err := os.OpenFile(logfileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Cannot open logfile %s for appending Factorio Server output: %s", logfileName, err)
		return err
	}
	defer file.Close()

	logline = logline + "\n"

	if _, err = file.WriteString(logline); err != nil {
		log.Printf("Error appending to %s: %s", logfileName, err)
		return err
	}

	return nil
}

func (server *Server) checkLogError(logline []string) error {
	// TODO Handle errors generated by running Factorio Server
	log.Println(logline)

	return nil
}

func init() {
	websocket.WebsocketHub.RegisterControlHandler <- serverWebsocketControl
}

// react to websocket control messages and run the command if it is requested
func serverWebsocketControl(controls websocket.WsControls) {
	log.Println(controls)
	if controls.Type == "command" {
		command := controls.Value
		server := GetFactorioServer()
		if server.GetRunning() {
			log.Printf("Received command: %v", command)

			err := server.SendRCON(command)
			if err != nil {
				log.Printf("Error sending rcon command: %s", err)
				return
			}

			log.Printf("Command sent to Factorio: %s", command)
		}
	}
}

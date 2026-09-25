package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

var managerStartedAt = time.Now()

type DiskMetrics struct {
	Path           string  `json:"path"`
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
	Available      bool    `json:"available"`
}

type HostMetrics struct {
	CPUPercent          *float64    `json:"cpu_percent"`
	LogicalCPUs         int         `json:"logical_cpus"`
	Load1               float64     `json:"load_1"`
	Load5               float64     `json:"load_5"`
	Load15              float64     `json:"load_15"`
	UptimeSeconds       float64     `json:"uptime_seconds"`
	MemoryTotalBytes    uint64      `json:"memory_total_bytes"`
	MemoryUsedBytes     uint64      `json:"memory_used_bytes"`
	ContainerUsedBytes  uint64      `json:"container_memory_used_bytes"`
	ContainerLimitBytes uint64      `json:"container_memory_limit_bytes"`
	Disk                DiskMetrics `json:"data_disk"`
}

type ManagerMetrics struct {
	UptimeSeconds int64    `json:"uptime_seconds"`
	CPUPercent    *float64 `json:"cpu_percent"`
	RSSBytes      uint64   `json:"rss_bytes"`
	GoMemoryBytes uint64   `json:"go_memory_bytes"`
	Goroutines    int      `json:"goroutines"`
}

type MetricAvailability struct {
	Available bool   `json:"available"`
	Source    string `json:"source"`
	Reason    string `json:"reason,omitempty"`
}

type FactorioMetrics struct {
	Running          bool                     `json:"running"`
	State            string                   `json:"state"`
	ProcessID        int                      `json:"process_id"`
	CPUPercent       *float64                 `json:"cpu_percent"`
	RSSBytes         uint64                   `json:"rss_bytes"`
	UptimeSeconds    int64                    `json:"uptime_seconds"`
	RCONConnected    bool                     `json:"rcon_connected"`
	Savefile         string                   `json:"savefile"`
	Version          string                   `json:"version"`
	PlayersOnline    int                      `json:"players_online"`
	MaxPlayers       int                      `json:"max_players"`
	Players          []factorio.RuntimePlayer `json:"players"`
	PlayerDataSource string                   `json:"player_data_source"`
	UPS              *float64                 `json:"ups"`
	UPSAvailability  MetricAvailability       `json:"ups_availability"`
}

type SystemMetricsResponse struct {
	Timestamp time.Time       `json:"timestamp"`
	Host      HostMetrics     `json:"host"`
	Manager   ManagerMetrics  `json:"manager"`
	Factorio  FactorioMetrics `json:"factorio"`

	// Legacy fields remain available for clients built before the monitoring page.
	ManagerUptime      int64   `json:"manager_uptime_seconds"`
	SystemUptime       float64 `json:"system_uptime_seconds"`
	MemoryTotalBytes   uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64  `json:"memory_used_bytes"`
	ProcessMemoryBytes uint64  `json:"process_memory_bytes"`
	Load1              float64 `json:"load_1"`
	Load5              float64 `json:"load_5"`
	Load15             float64 `json:"load_15"`
	Goroutines         int     `json:"goroutines"`
	FactorioRunning    bool    `json:"factorio_running"`
}

type cpuSnapshot struct {
	total, idle, manager, factorio uint64
	factorioPID                    int
}

var metricSampler = struct {
	sync.Mutex
	previous *cpuSnapshot
}{}

func parseFirstFloat(path string) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return value
}

func memoryFromScanner(scanner *bufio.Scanner) (total, available uint64) {
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			total = value * 1024
		case "MemAvailable":
			available = value * 1024
		}
	}
	return total, available
}

func linuxMemory() (total, available uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	return memoryFromScanner(bufio.NewScanner(file))
}

func loadAverages() (float64, float64, float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	one, _ := strconv.ParseFloat(fields[0], 64)
	five, _ := strconv.ParseFloat(fields[1], 64)
	fifteen, _ := strconv.ParseFloat(fields[2], 64)
	return one, five, fifteen
}

func readUintFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return value
}

func cgroupMemory() (used, limit uint64) {
	used = readUintFile("/sys/fs/cgroup/memory.current")
	limit = readUintFile("/sys/fs/cgroup/memory.max")
	if limit > 1<<62 {
		limit = 0
	}
	return used, limit
}

func parseCPUStat(value string) (total, idle uint64, err error) {
	fields := strings.Fields(value)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, errors.New("invalid /proc/stat cpu line")
	}
	for index, field := range fields[1:] {
		part, parseErr := strconv.ParseUint(field, 10, 64)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		total += part
		if index == 3 || index == 4 {
			idle += part
		}
	}
	return total, idle, nil
}

func readHostCPU() (total, idle uint64) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0
	}
	total, idle, _ = parseCPUStat(scanner.Text())
	return total, idle
}

func readProcessCPU(pid int) uint64 {
	if pid <= 0 {
		return 0
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0
	}
	closing := strings.LastIndex(string(data), ")")
	if closing < 0 {
		return 0
	}
	fields := strings.Fields(string(data)[closing+1:])
	if len(fields) < 13 {
		return 0
	}
	user, _ := strconv.ParseUint(fields[11], 10, 64)
	system, _ := strconv.ParseUint(fields[12], 10, 64)
	return user + system
}

func percent(value float64) *float64 {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return &value
}

func sampleCPU(factorioPID int) (host, manager, factorioCPU *float64) {
	total, idle := readHostCPU()
	current := cpuSnapshot{total: total, idle: idle, manager: readProcessCPU(os.Getpid()), factorio: readProcessCPU(factorioPID), factorioPID: factorioPID}
	metricSampler.Lock()
	previous := metricSampler.previous
	metricSampler.previous = &current
	metricSampler.Unlock()
	if previous == nil || current.total <= previous.total {
		return nil, nil, nil
	}
	totalDelta := float64(current.total - previous.total)
	idleDelta := float64(current.idle - previous.idle)
	host = percent((totalDelta - idleDelta) / totalDelta * 100)
	if current.manager >= previous.manager {
		manager = percent(float64(current.manager-previous.manager) / totalDelta * 100)
	}
	if factorioPID > 0 && previous.factorioPID == factorioPID && current.factorio >= previous.factorio {
		factorioCPU = percent(float64(current.factorio-previous.factorio) / totalDelta * 100)
	}
	return host, manager, factorioCPU
}

func configuredMaxPlayers() int {
	data, err := os.ReadFile(bootstrap.GetConfig().SettingsFile)
	if err != nil {
		return 0
	}
	var settings struct {
		MaxPlayers int `json:"max_players"`
	}
	if json.Unmarshal(data, &settings) != nil {
		return 0
	}
	return settings.MaxPlayers
}

func CollectSystemMetrics() SystemMetricsResponse {
	total, available := linuxMemory()
	one, five, fifteen := loadAverages()
	containerUsed, containerLimit := cgroupMemory()
	var goMemory runtime.MemStats
	runtime.ReadMemStats(&goMemory)
	server := factorio.GetFactorioServer()
	status := server.Status()
	runtimeMetrics := server.RuntimeMetrics()
	hostCPU, managerCPU, factorioCPU := sampleCPU(runtimeMetrics.ProcessID)
	managerUptime := int64(time.Since(managerStartedAt).Seconds())
	managerRSS := processRSSBytes(os.Getpid())
	factorioRSS := processRSSBytes(runtimeMetrics.ProcessID)
	memoryUsed := uint64(0)
	if total > available {
		memoryUsed = total - available
	}
	disk := platformDiskMetrics(bootstrap.GetConfig().FactorioSavesDir)
	response := SystemMetricsResponse{
		Timestamp: time.Now().UTC(),
		Host: HostMetrics{
			CPUPercent: hostCPU, LogicalCPUs: runtime.NumCPU(), Load1: one, Load5: five, Load15: fifteen,
			UptimeSeconds: parseFirstFloat("/proc/uptime"), MemoryTotalBytes: total, MemoryUsedBytes: memoryUsed,
			ContainerUsedBytes: containerUsed, ContainerLimitBytes: containerLimit, Disk: disk,
		},
		Manager: ManagerMetrics{UptimeSeconds: managerUptime, CPUPercent: managerCPU, RSSBytes: managerRSS, GoMemoryBytes: goMemory.Sys, Goroutines: runtime.NumGoroutine()},
		Factorio: FactorioMetrics{
			Running: status.Running, State: status.State, ProcessID: runtimeMetrics.ProcessID, CPUPercent: factorioCPU,
			RSSBytes: factorioRSS, UptimeSeconds: runtimeMetrics.UptimeSeconds, RCONConnected: status.RconConnected,
			Savefile: status.Savefile, Version: strings.TrimSuffix(status.Version.String(), ".0"), PlayersOnline: runtimeMetrics.OnlineCount,
			MaxPlayers: configuredMaxPlayers(), Players: runtimeMetrics.OnlinePlayers, PlayerDataSource: runtimeMetrics.PlayerDataSource,
			UPSAvailability: MetricAvailability{Available: false, Source: "not-available", Reason: "Factorio does not expose live UPS through a non-cheat native command"},
		},
		ManagerUptime: managerUptime, SystemUptime: parseFirstFloat("/proc/uptime"), MemoryTotalBytes: total,
		MemoryUsedBytes: memoryUsed, ProcessMemoryBytes: goMemory.Sys, Load1: one, Load5: five, Load15: fifteen,
		Goroutines: runtime.NumGoroutine(), FactorioRunning: status.Running,
	}
	return response
}

func SystemMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_ = json.NewEncoder(w).Encode(CollectSystemMetrics())
}

func StartMetricsBroadcaster() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics := CollectSystemMetrics()
			websocket.WebsocketHub.GetRoom("system_metrics").Send(metrics)
			monitorSystemThresholds(metrics)
		}
	}()
}

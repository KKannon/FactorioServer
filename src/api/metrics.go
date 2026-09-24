package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

var managerStartedAt = time.Now()

type SystemMetricsResponse struct {
	Timestamp          time.Time `json:"timestamp"`
	ManagerUptime      int64     `json:"manager_uptime_seconds"`
	SystemUptime       float64   `json:"system_uptime_seconds"`
	MemoryTotalBytes   uint64    `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64    `json:"memory_used_bytes"`
	ProcessMemoryBytes uint64    `json:"process_memory_bytes"`
	Load1              float64   `json:"load_1"`
	Load5              float64   `json:"load_5"`
	Load15             float64   `json:"load_15"`
	Goroutines         int       `json:"goroutines"`
	FactorioRunning    bool      `json:"factorio_running"`
}

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

func linuxMemory() (total, available uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
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

func SystemMetrics(w http.ResponseWriter, _ *http.Request) {
	total, available := linuxMemory()
	one, five, fifteen := loadAverages()
	var process runtime.MemStats
	runtime.ReadMemStats(&process)
	response := SystemMetricsResponse{
		Timestamp: time.Now().UTC(), ManagerUptime: int64(time.Since(managerStartedAt).Seconds()),
		SystemUptime: parseFirstFloat("/proc/uptime"), MemoryTotalBytes: total,
		ProcessMemoryBytes: process.Sys, Load1: one, Load5: five, Load15: fifteen,
		Goroutines: runtime.NumGoroutine(), FactorioRunning: factorio.GetFactorioServer().GetRunning(),
	}
	if total > available {
		response.MemoryUsedBytes = total - available
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	json.NewEncoder(w).Encode(response)
}

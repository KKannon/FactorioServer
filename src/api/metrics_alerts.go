package api

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var monitoringAlertState = struct {
	sync.Mutex
	active map[string]bool
}{active: make(map[string]bool)}

func monitoringThreshold(name string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(name)), 64)
	if err != nil || value < 1 || value > 100 {
		return fallback
	}
	return value
}

func claimMonitoringAlert(key string, current, threshold float64) bool {
	monitoringAlertState.Lock()
	defer monitoringAlertState.Unlock()
	if current >= threshold && !monitoringAlertState.active[key] {
		monitoringAlertState.active[key] = true
		return true
	}
	if current <= threshold-5 {
		monitoringAlertState.active[key] = false
	}
	return false
}

func queueMonitoringAlert(recipient, resource string) {
	if notifications == nil || strings.TrimSpace(recipient) == "" {
		return
	}
	payload := notificationPayload{
		Event: "MonitoringAlert", Recipient: recipient, ActorName: "Monitoramento automático", ActorRole: "system",
		Resource: resource, OccurredAt: time.Now().UTC().Format(time.RFC3339), IdempotencyKey: notificationID("MonitoringAlert"),
	}
	select {
	case notifications.queue <- payload:
	default:
	}
}

func monitorSystemThresholds(metrics SystemMetricsResponse) {
	recipient := strings.TrimSpace(os.Getenv("STUPID_MONITORING_ALERT_RECIPIENT"))
	if recipient == "" {
		return
	}
	diskThreshold := monitoringThreshold("STUPID_MONITORING_DISK_THRESHOLD", 90)
	if metrics.Host.Disk.Available && claimMonitoringAlert("disk", metrics.Host.Disk.UsedPercent, diskThreshold) {
		queueMonitoringAlert(recipient, fmt.Sprintf("Disco dos dados do Factorio em %.1f%%", metrics.Host.Disk.UsedPercent))
	}
	memoryThreshold := monitoringThreshold("STUPID_MONITORING_MEMORY_THRESHOLD", 90)
	used, total := metrics.Host.ContainerUsedBytes, metrics.Host.ContainerLimitBytes
	if total == 0 {
		used, total = metrics.Host.MemoryUsedBytes, metrics.Host.MemoryTotalBytes
	}
	if total > 0 {
		memoryPercent := float64(used) / float64(total) * 100
		if claimMonitoringAlert("memory", memoryPercent, memoryThreshold) {
			queueMonitoringAlert(recipient, fmt.Sprintf("Memória do servidor Factorio em %.1f%%", memoryPercent))
		}
	}
}

//go:build linux

package api

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func platformDiskMetrics(path string) DiskMetrics {
	metrics := DiskMetrics{Path: path}
	var info syscall.Statfs_t
	if syscall.Statfs(path, &info) != nil {
		return metrics
	}
	metrics.TotalBytes = info.Blocks * uint64(info.Bsize)
	metrics.AvailableBytes = info.Bavail * uint64(info.Bsize)
	if metrics.TotalBytes > metrics.AvailableBytes {
		metrics.UsedBytes = metrics.TotalBytes - metrics.AvailableBytes
		metrics.UsedPercent = float64(metrics.UsedBytes) / float64(metrics.TotalBytes) * 100
	}
	metrics.Available = true
	return metrics
}

func processRSSBytes(pid int) uint64 {
	if pid <= 0 {
		return 0
	}
	file, err := os.Open("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "VmRSS:" {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value * 1024
		}
	}
	return 0
}

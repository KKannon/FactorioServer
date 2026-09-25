//go:build !linux

package api

func platformDiskMetrics(path string) DiskMetrics {
	return DiskMetrics{Path: path, Available: false}
}

func processRSSBytes(_ int) uint64 {
	return 0
}

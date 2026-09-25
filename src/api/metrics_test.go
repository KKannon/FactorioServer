package api

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseCPUStat(t *testing.T) {
	total, idle, err := parseCPUStat("cpu  100 20 30 400 50 6 7 8 0 0")
	if err != nil {
		t.Fatal(err)
	}
	if total != 621 || idle != 450 {
		t.Fatalf("unexpected cpu totals: total=%d idle=%d", total, idle)
	}
}

func TestMemoryFromScanner(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("MemTotal: 1024 kB\nMemAvailable: 256 kB\n"))
	total, available := memoryFromScanner(scanner)
	if total != 1024*1024 || available != 256*1024 {
		t.Fatalf("unexpected memory values: total=%d available=%d", total, available)
	}
}

func TestMonitoringAlertUsesHysteresis(t *testing.T) {
	monitoringAlertState.Lock()
	monitoringAlertState.active = make(map[string]bool)
	monitoringAlertState.Unlock()
	if !claimMonitoringAlert("disk-test", 91, 90) {
		t.Fatal("expected first threshold crossing to alert")
	}
	if claimMonitoringAlert("disk-test", 93, 90) {
		t.Fatal("active alert must not repeat")
	}
	if claimMonitoringAlert("disk-test", 85, 90) {
		t.Fatal("recovery must not send an alert")
	}
	if !claimMonitoringAlert("disk-test", 92, 90) {
		t.Fatal("expected a new alert after recovery")
	}
}

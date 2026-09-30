package api

import "testing"

func TestValidFactorioClientUpdateFilename(t *testing.T) {
	valid := []string{"latest.yml", "Factorio-Client-0.2.4.exe", "Factorio-Client-Setup-0.2.4.exe", "Factorio-Client-Setup-0.2.4.exe.blockmap", "Factorio-Launcher-0.3.0.exe", "Factorio-Launcher-Setup-0.3.0.exe", "Factorio-Launcher-Setup-0.3.0.exe.blockmap"}
	for _, name := range valid {
		if !validFactorioClientUpdateFilename(name) {
			t.Fatalf("expected %q to be accepted", name)
		}
	}
	invalid := []string{"../conf.json", "Factorio-Client-latest.exe", "Factorio-Client-0.2.exe", "source.zip", "latest.json"}
	for _, name := range invalid {
		if validFactorioClientUpdateFilename(name) {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

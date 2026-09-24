package factorio

import "testing"

func TestValidInstallVersion(t *testing.T) {
	for _, value := range []string{"stable", "latest", "2.0.77", " 1.1.110 "} {
		if !ValidInstallVersion(value) {
			t.Errorf("ValidInstallVersion(%q) = false", value)
		}
	}
	for _, value := range []string{"", "2.0", "2.0.77.0", "../../tmp", "stable?x=1", "experimental"} {
		if ValidInstallVersion(value) {
			t.Errorf("ValidInstallVersion(%q) = true", value)
		}
	}
}

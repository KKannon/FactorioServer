package factorio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddAccessEntryIsIdempotentAndSorted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.json")
	added, err := addAccessEntry(path, "zeta")
	if err != nil || !added {
		t.Fatalf("first add = %v, %v; want true, nil", added, err)
	}
	if _, err := addAccessEntry(path, "Alpha"); err != nil {
		t.Fatal(err)
	}
	added, err = addAccessEntry(path, "alpha")
	if err != nil || added {
		t.Fatalf("duplicate add = %v, %v; want false, nil", added, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != "Alpha" || values[1] != "zeta" {
		t.Fatalf("access list = %#v; want [Alpha zeta]", values)
	}
}

func TestValidPlayerName(t *testing.T) {
	for _, value := range []string{"Julio", "player_01", "a-b.c"} {
		if !ValidPlayerName(value) {
			t.Errorf("ValidPlayerName(%q) = false", value)
		}
	}
	for _, value := range []string{"", "a", "has space", "unsafe/entry"} {
		if ValidPlayerName(value) {
			t.Errorf("ValidPlayerName(%q) = true", value)
		}
	}
}

func TestRemoveAccessEntryIgnoresCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admins.json")
	if err := os.WriteFile(path, []byte(`["Alpha","Beta"]`), 0600); err != nil {
		t.Fatal(err)
	}
	removed, err := removeAccessEntry(path, "alpha")
	if err != nil || !removed {
		t.Fatalf("remove = %v, %v; want true, nil", removed, err)
	}
	values, err := readAccessList(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0] != "Beta" {
		t.Fatalf("access list = %#v; want [Beta]", values)
	}
}

func TestWhitelistPolicyDefaultsToEnabled(t *testing.T) {
	enabled, err := readWhitelistEnabledAt(filepath.Join(t.TempDir(), "missing-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("missing policy must default to enabled")
	}
}

func TestWhitelistPolicyReadsDisabledState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{"whitelist_enabled":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	enabled, err := readWhitelistEnabledAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("stored disabled state was ignored")
	}
}

func TestListBansSupportsUsersAndAddresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server-banlist.json")
	data := `[{"username":"Player","reason":"spam"},{"address":"192.0.2.4","reason":"abuse"}]`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	bans, err := listBans(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(bans) != 2 || bans[0].Username != "Player" || bans[1].Address != "192.0.2.4" {
		t.Fatalf("unexpected bans: %#v", bans)
	}
}

func TestBanReasonRejectsControlCharacters(t *testing.T) {
	if validBanReason("line one\nline two") {
		t.Fatal("newline must not be accepted in an RCON command")
	}
	if !validBanReason("griefing and spam") {
		t.Fatal("plain text reason should be accepted")
	}
}

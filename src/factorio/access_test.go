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

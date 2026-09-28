package factorio

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBuildPlayerBridgeArchiveContainsFactorioMod(t *testing.T) {
	archive, filename, err := PlayerBridgeArchive()
	if err != nil {
		t.Fatal(err)
	}
	if filename != "factorio-server-manager-bridge_1.0.1.zip" {
		t.Fatalf("unexpected bridge filename %q", filename)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	prefix := PlayerBridgeName + "_" + playerBridgeVersion + "/"
	found := map[string]bool{}
	for _, file := range reader.File {
		found[file.Name] = true
		if file.Name == prefix+"control.lua" {
			contents, openErr := file.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			control, readErr := io.ReadAll(contents)
			_ = contents.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Contains(control, []byte("character.max_health")) || bytes.Contains(control, []byte("character.prototype.max_health")) {
				t.Fatal("bridge must use the Factorio 2.x LuaEntity max_health API")
			}
		}
	}
	for _, required := range []string{prefix + "info.json", prefix + "control.lua"} {
		if !found[required] {
			t.Fatalf("bridge archive does not contain %s", required)
		}
	}
}

func TestParsePlayerSnapshotRequiresBridgeMarkerAndSchema(t *testing.T) {
	response := "command output\n" + playerBridgeResponse + `{"schema_version":1,"generated_tick":120,"source":"factorio-server-manager-bridge","players":[{"name":"Alice","online_ticks":60}],"capabilities":{"inventory":true}}`
	snapshot, err := parsePlayerSnapshot(response)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.GeneratedTick != 120 || len(snapshot.Players) != 1 || snapshot.Players[0].Name != "Alice" || !snapshot.Capabilities.Inventory {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if _, err := parsePlayerSnapshot(strings.TrimPrefix(response, "command output\n"+playerBridgeResponse)); err == nil {
		t.Fatal("response without bridge marker must be rejected")
	}
}

func TestVisiblePlayerIntelligenceRestrictsMembersToExactUsername(t *testing.T) {
	snapshot := PlayerIntelligenceSnapshot{Players: []PlayerIntelligence{{Name: "Alice"}, {Name: "Bob"}}}
	visible := VisiblePlayerIntelligence(snapshot, "Bob", false)
	if len(visible) != 1 || visible[0].Name != "Bob" {
		t.Fatalf("member received unexpected profiles: %#v", visible)
	}
	if visible := VisiblePlayerIntelligence(snapshot, "bob", false); len(visible) != 0 {
		t.Fatalf("username matching must be exact: %#v", visible)
	}
	if visible := VisiblePlayerIntelligence(snapshot, "", true); len(visible) != 2 {
		t.Fatalf("manager received %d profiles, want 2", len(visible))
	}
}

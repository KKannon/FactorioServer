package factorio

import "testing"

func TestParseModDependencyKinds(t *testing.T) {
	tests := []struct{ raw, name, kind, operator, version string }{
		{"base >= 2.0", "base", "required", ">=", "2.0"},
		{"? helper >= 1.2.3", "helper", "optional", ">=", "1.2.3"},
		{"(?) hidden", "hidden", "hidden-optional", "", ""},
		{"! conflict", "conflict", "incompatible", "", ""},
		{"~ ordering = 3.1", "ordering", "no-load-order", "==", "3.1"},
	}
	for _, test := range tests {
		dependency, err := parseModDependency(test.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", test.raw, err)
		}
		if dependency.Name != test.name || dependency.Kind != test.kind || dependency.Operator != test.operator || dependency.VersionText != test.version {
			t.Fatalf("unexpected dependency for %q: %#v", test.raw, dependency)
		}
	}
}

func TestDependencyStatusDetectsMissingDisabledVersionAndConflict(t *testing.T) {
	SetFactorioServer(Server{Version: Version{2, 0, 77}, BaseModVersion: "2.0.77"})
	mods := Mods{
		ModSimpleList: ModSimpleList{Mods: []ModSimple{{Name: "base", Enabled: true}, {Name: "helper", Enabled: false}, {Name: "conflict", Enabled: true}}},
		ModInfoList: ModInfoList{Mods: []ModInfo{
			{Name: "subject", Compatibility: true, Dependencies: []string{"missing >= 1.0", "helper >= 2.0", "! conflict"}},
			{Name: "helper", Version: "1.0", Compatibility: true},
			{Name: "conflict", Version: "1.0", Compatibility: true},
		}},
	}
	result := mods.ListInstalledMods().ModsResult[0]
	if result.Compatibility {
		t.Fatal("expected incompatible dependency graph")
	}
	want := []string{"missing", "disabled", "conflict"}
	for index, state := range want {
		if result.DependencyStatus[index].State != state {
			t.Fatalf("dependency %d: want %s, got %s", index, state, result.DependencyStatus[index].State)
		}
	}
}

func TestOptionalMissingDependencyIsCompatible(t *testing.T) {
	SetFactorioServer(Server{Version: Version{2, 0, 77}, BaseModVersion: "2.0.77"})
	mods := Mods{
		ModSimpleList: ModSimpleList{Mods: []ModSimple{{Name: "base", Enabled: true}}},
		ModInfoList:   ModInfoList{Mods: []ModInfo{{Name: "subject", Compatibility: true, Dependencies: []string{"? absent >= 9.0"}}}},
	}
	result := mods.ListInstalledMods().ModsResult[0]
	if !result.Compatibility || !result.DependencyStatus[0].Satisfied {
		t.Fatal("missing optional dependency should remain compatible")
	}
}

func TestEnabledRequiredByIgnoresOptionalAndDisabledMods(t *testing.T) {
	mods := Mods{
		ModSimpleList: ModSimpleList{Mods: []ModSimple{{Name: "required", Enabled: true}, {Name: "optional", Enabled: true}, {Name: "disabled", Enabled: false}}},
		ModInfoList: ModInfoList{Mods: []ModInfo{
			{Name: "required", Dependencies: []string{"library >= 1.0"}},
			{Name: "optional", Dependencies: []string{"? library"}},
			{Name: "disabled", Dependencies: []string{"library"}},
		}},
	}
	requiredBy := mods.EnabledRequiredBy("library")
	if len(requiredBy) != 1 || requiredBy[0] != "required" {
		t.Fatalf("unexpected reverse dependencies: %#v", requiredBy)
	}
}

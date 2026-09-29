package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestMod(t *testing.T, directory, name, version string) string {
	t.Helper()
	filename := name + "_" + version + ".zip"
	file, err := os.Create(filepath.Join(directory, filename))
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create(name + "_" + version + "/info.json")
	if err != nil {
		t.Fatal(err)
	}
	metadata, _ := json.Marshal(map[string]interface{}{
		"name": name, "version": version, "title": name, "author": "test",
		"factorio_version": "1.1", "dependencies": []string{"base >= 1.1"},
	})
	if _, err = entry.Write(metadata); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func TestPlayerModBundleContainsOnlyEnabledArchives(t *testing.T) {
	directory := t.TempDir()
	enabledFile := writeTestMod(t, directory, "enabled-mod", "1.0.0")
	_ = writeTestMod(t, directory, "disabled-mod", "1.0.0")
	modList := []byte(`{"mods":[{"name":"base","enabled":true},{"name":"enabled-mod","enabled":true},{"name":"disabled-mod","enabled":false}]}`)
	if err := os.WriteFile(filepath.Join(directory, "mod-list.json"), modList, 0644); err != nil {
		t.Fatal(err)
	}

	plan, err := preparePlayerModBundle(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Status.Required || plan.Status.Count != 1 || filepath.Base(plan.Files[0]) != enabledFile {
		t.Fatalf("unexpected bundle plan: %+v", plan)
	}

	var output bytes.Buffer
	if err = writePlayerModBundle(&output, directory, plan); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, file := range reader.File {
		names[file.Name] = true
	}
	if !names[enabledFile] || !names["mod-list.json"] || !names["LEIA-ME.txt"] || names["disabled-mod_1.0.0.zip"] {
		t.Fatalf("unexpected archive contents: %+v", names)
	}
}

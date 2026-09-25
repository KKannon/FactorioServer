package factorio

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestSave(t *testing.T, path, value string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("world/control.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write([]byte(value)); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func readTestSave(t *testing.T, path string) string {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	reader, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestBackupNameRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 9, 24, 18, 30, 0, 123, time.UTC)
	name, err := backupName("my-world.zip", createdAt)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := parseBackupName(name)
	if err != nil {
		t.Fatal(err)
	}
	if backup.SaveName != "my-world.zip" || !backup.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected parsed backup: %#v", backup)
	}
}

func TestRestoreCreatesSafetyBackupAndReplacesSave(t *testing.T) {
	root := t.TempDir()
	savesDir := filepath.Join(root, "saves")
	backupsDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(savesDir, 0755); err != nil {
		t.Fatal(err)
	}
	savePath := filepath.Join(savesDir, "world.zip")
	writeTestSave(t, savePath, "old world")
	backup, err := createBackupAt(savesDir, backupsDir, "world.zip", time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSave(t, savePath, "current world")

	restored, safety, err := restoreSaveBackupAt(savesDir, backupsDir, backup.Name, time.Date(2026, 9, 24, 19, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Name != backup.Name || safety == nil {
		t.Fatalf("unexpected restore result: %#v, %#v", restored, safety)
	}
	if got := readTestSave(t, savePath); got != "old world" {
		t.Fatalf("restored save = %q; want old world", got)
	}
	safetyPath := filepath.Join(backupsDir, safety.Name)
	if got := readTestSave(t, safetyPath); got != "current world" {
		t.Fatalf("safety backup = %q; want current world", got)
	}
}

func TestRestoreRejectsInvalidBackupArchive(t *testing.T) {
	root := t.TempDir()
	savesDir := filepath.Join(root, "saves")
	backupsDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}
	name, _ := backupName("world.zip", time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC))
	if err := os.WriteFile(filepath.Join(backupsDir, name), []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := restoreSaveBackupAt(savesDir, backupsDir, name, time.Now()); err == nil {
		t.Fatal("invalid backup archive was accepted")
	}
}

func TestDuplicateBackupNeverRemovesExistingCopy(t *testing.T) {
	root := t.TempDir()
	savesDir := filepath.Join(root, "saves")
	backupsDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(savesDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSave(t, filepath.Join(savesDir, "world.zip"), "original")
	createdAt := time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC)
	backup, err := createBackupAt(savesDir, backupsDir, "world.zip", createdAt)
	if err != nil {
		t.Fatal(err)
	}
	writeTestSave(t, filepath.Join(savesDir, "world.zip"), "replacement")
	if _, err = createBackupAt(savesDir, backupsDir, "world.zip", createdAt); err == nil {
		t.Fatal("duplicate backup name was accepted")
	}
	if got := readTestSave(t, filepath.Join(backupsDir, backup.Name)); got != "original" {
		t.Fatalf("existing backup changed to %q", got)
	}
}

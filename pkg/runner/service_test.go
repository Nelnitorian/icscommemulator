package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configPath, "dummy.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := cleanConfigDirectory(configPath); err != nil {
		t.Fatalf("cleanConfigDirectory() error: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config dir to be removed")
	}
}

func TestCleanConfigDirectoryRejectsUnsafePaths(t *testing.T) {
	if err := cleanConfigDirectory("/tmp"); err == nil {
		t.Fatalf("expected error for /tmp path")
	}
	if err := cleanConfigDirectory("/"); err == nil {
		t.Fatalf("expected error for / path")
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIEC104MasterLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "master.yaml")
	config := `protocol: iec104
messages: []`

	if err := os.WriteFile(cfgPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	master, err := NewIEC104Master(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if master.config.Protocol != "iec104" {
		t.Fatalf("expected protocol iec104, got %s", master.config.Protocol)
	}

	if err := master.setup(); err != nil {
		t.Fatalf("setup failed with empty schedule: %v", err)
	}
}

func TestParseBool(t *testing.T) {
	if !parseBool("true") {
		t.Fatal("expected true")
	}
	if parseBool("false") {
		t.Fatal("expected false")
	}
	if !parseBool("1") {
		t.Fatal("expected true for 1")
	}
}

func TestParseFloat(t *testing.T) {
	if got := parseFloat("1.5"); got != 1.5 {
		t.Fatalf("expected 1.5, got %v", got)
	}
	if got := parseFloat("bad"); got != 0 {
		t.Fatalf("expected 0 for invalid input, got %v", got)
	}
}

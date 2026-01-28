package appconfig

import (
	"os"
	"path/filepath"
	"testing"

	"icscommemulator/pkg/scenario"
)

func TestLoadConfigYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := []byte(`
scenarios:
  - name: demo_yaml
    path: ./demo.yaml
    overwrite: true
`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(cfg.Scenarios))
	}
	if cfg.Scenarios[0].Name != "demo_yaml" {
		t.Fatalf("unexpected scenario name: %s", cfg.Scenarios[0].Name)
	}
}

func TestLoadConfigJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := []byte(`{"scenarios":[{"name":"demo_json","path":"./demo.json"}]}`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(cfg.Scenarios))
	}
	if cfg.Scenarios[0].Name != "demo_json" {
		t.Fatalf("unexpected scenario name: %s", cfg.Scenarios[0].Name)
	}
}

func TestImportScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yaml")
	scenarioContent := []byte(`
protocol: modbus
ip_network: 192.168.1.0/24
nodes:
  - id: master_0
    role: master
    name: master_0
    ip: 192.168.1.10
  - id: slave_0
    role: slave
    name: slave_0
    ip: 192.168.1.11
    port: 502
    slave_id: 1
edges: []
`)

	if err := os.WriteFile(scenarioPath, scenarioContent, 0644); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	cfg := &Config{
		Scenarios: []ScenarioImport{
			{
				Name:      "import_test",
				Path:      "scenario.yaml",
				Overwrite: true,
			},
		},
	}

	if err := cfg.ImportScenarios(scenario.NewStorage(), tmpDir); err != nil {
		t.Fatalf("ImportScenarios() error: %v", err)
	}

	defer scenario.DeleteScenario("import_test")

	if !scenario.CheckScenarioExists("import_test") {
		t.Fatalf("scenario not imported")
	}
}

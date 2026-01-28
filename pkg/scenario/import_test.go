package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportScenarioFileFromFlatYAML(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "flat.yaml")
	content := []byte(`
protocol: modbus
ip_network: 192.168.10.0/24
nodes:
  - id: master_0
    role: master
    name: master_0
    ip: 192.168.10.10
  - id: slave_0
    role: slave
    name: slave_0
    ip: 192.168.10.11
    port: 502
    slave_id: 1
edges: []
`)

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	if err := ImportScenarioFile("flat_import", filePath); err != nil {
		t.Fatalf("ImportScenarioFile() error: %v", err)
	}
	defer DeleteScenario("flat_import")

	scenarioMap, err := GetCytoscapeScenarioAsMap("flat_import")
	if err != nil {
		t.Fatalf("failed to read imported scenario: %v", err)
	}

	nodes, ok := scenarioMap["nodes"].([]interface{})
	if !ok || len(nodes) == 0 {
		t.Fatalf("expected nodes in imported scenario")
	}

	firstNode, ok := nodes[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected node to be a map")
	}

	if _, ok := firstNode["data"]; !ok {
		t.Fatalf("expected nodes to be normalized to cytoscape format")
	}
}

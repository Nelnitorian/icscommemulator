package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDockerNetworkInterfaceAndConfig(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	content := []byte(`
version: "3.8"
networks:
  ics_net:
    driver: bridge
    ipam:
      config:
        - subnet: 192.168.10.0/24
`)
	if err := os.WriteFile(composePath, content, 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	r := NewRunner()
	r.filePath = composePath

	name, err := r.GetDockerNetworkInterface()
	if err != nil {
		t.Fatalf("GetDockerNetworkInterface() error: %v", err)
	}
	if name != "ics_net" {
		t.Fatalf("expected network name ics_net, got %s", name)
	}

	netName, subnet, err := r.GetDockerNetworkConfig()
	if err != nil {
		t.Fatalf("GetDockerNetworkConfig() error: %v", err)
	}
	if netName != "ics_net" || subnet != "192.168.10.0/24" {
		t.Fatalf("unexpected network config: %s %s", netName, subnet)
	}
}

package docker

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("/tmp/config", "modbus")

	if gen.configPath != "/tmp/config" {
		t.Errorf("Expected configPath '/tmp/config', got '%s'", gen.configPath)
	}

	if gen.protocol != "modbus" {
		t.Errorf("Expected protocol 'modbus', got '%s'", gen.protocol)
	}

	if gen.compose.Version != "3.8" {
		t.Errorf("Expected Docker Compose version '3.8', got '%s'", gen.compose.Version)
	}
}

func TestAddNetwork(t *testing.T) {
	gen := NewGenerator("/tmp/config", "modbus")
	gen.AddNetwork("test_network", "192.168.1.0/24")

	network, exists := gen.compose.Networks["test_network"]
	if !exists {
		t.Fatal("Network 'test_network' was not added")
	}

	if network.Driver != "bridge" {
		t.Errorf("Expected driver 'bridge', got '%s'", network.Driver)
	}

	if len(network.IPAM.Config) == 0 {
		t.Fatal("IPAM config is empty")
	}

	if network.IPAM.Config[0].Subnet != "192.168.1.0/24" {
		t.Errorf("Expected subnet '192.168.1.0/24', got '%s'", network.IPAM.Config[0].Subnet)
	}
}

func TestAddMasterService(t *testing.T) {
	gen := NewGenerator("/tmp/config", "modbus")
	gen.AddNetwork("ics_network", "192.168.1.0/24")

	err := gen.AddMasterService(0, "192.168.1.10", "ics_network", nil)
	if err != nil {
		t.Fatalf("AddMasterService() error = %v", err)
	}

	service, exists := gen.compose.Services["master_0"]
	if !exists {
		t.Fatal("Master service was not added")
	}

	if service.Container != "master_0" {
		t.Errorf("Expected container name 'master_0', got '%s'", service.Container)
	}

	if service.Build == nil {
		t.Fatal("Build config is nil")
	}

	expectedContext := "./protocols/modbus/master"
	if service.Build.Context != expectedContext {
		t.Errorf("Expected build context '%s', got '%s'", expectedContext, service.Build.Context)
	}
}

func TestAddSlaveService(t *testing.T) {
	gen := NewGenerator("/tmp/config", "modbus")
	gen.AddNetwork("ics_network", "192.168.1.0/24")

	err := gen.AddSlaveService(0, "192.168.1.20", "ics_network", 502, nil)
	if err != nil {
		t.Fatalf("AddSlaveService() error = %v", err)
	}

	service, exists := gen.compose.Services["slave_0"]
	if !exists {
		t.Fatal("Slave service was not added")
	}

	if service.Container != "slave_0" {
		t.Errorf("Expected container name 'slave_0', got '%s'", service.Container)
	}

	if len(service.Expose) == 0 {
		t.Fatal("Expose ports not set")
	}

	if service.Expose[0] != "502" {
		t.Errorf("Expected exposed port '502', got '%s'", service.Expose[0])
	}

	if service.HealthCheck == nil {
		t.Fatal("Health check not configured")
	}
}

func TestAddSlaveServiceWithDependencies(t *testing.T) {
	gen := NewGenerator("/tmp/config", "modbus")
	gen.AddNetwork("ics_network", "192.168.1.0/24")

	// Add first slave without dependencies
	gen.AddSlaveService(0, "192.168.1.20", "ics_network", 502, nil)

	// Add second slave with dependency on first
	deps := []string{"slave_0"}
	err := gen.AddSlaveService(1, "192.168.1.21", "ics_network", 502, deps)
	if err != nil {
		t.Fatalf("AddSlaveService() with dependencies error = %v", err)
	}

	service := gen.compose.Services["slave_1"]
	if len(service.DependsOn) == 0 {
		t.Fatal("Dependencies not set")
	}

	if _, exists := service.DependsOn["slave_0"]; !exists {
		t.Error("Expected dependency on 'slave_0' not found")
	}
}

func TestGenerate(t *testing.T) {
	tmpFile := "/tmp/test-compose.yml"
	defer os.Remove(tmpFile)

	gen := NewGenerator("/tmp/config", "modbus")
	gen.AddNetwork("ics_network", "192.168.1.0/24")
	gen.AddMasterService(0, "192.168.1.10", "ics_network", nil)
	gen.AddSlaveService(0, "192.168.1.20", "ics_network", 502, nil)

	err := gen.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Docker Compose file was not created")
	}

	// Read and parse the generated file
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	var compose DockerCompose
	err = yaml.Unmarshal(data, &compose)
	if err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify structure
	if compose.Version != "3.8" {
		t.Errorf("Expected version '3.8', got '%s'", compose.Version)
	}

	if len(compose.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(compose.Services))
	}

	if len(compose.Networks) != 1 {
		t.Errorf("Expected 1 network, got %d", len(compose.Networks))
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
		wantErr  bool
	}{
		{"Int port", 502, 502, false},
		{"Float port", 502.0, 502, false},
		{"String port", "502", 502, false},
		{"Invalid string", "abc", 0, true},
		{"Nil port", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParsePort() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetNetworkSubnet(t *testing.T) {
	subnet, err := GetNetworkSubnet("192.168.1.0", 24)
	if err != nil {
		t.Fatalf("GetNetworkSubnet() error = %v", err)
	}

	expected := "192.168.1.0/24"
	if subnet != expected {
		t.Errorf("Expected subnet '%s', got '%s'", expected, subnet)
	}
}

func TestProtocolDockerfiles(t *testing.T) {
	protocols := []string{"modbus", "dnp3", "iec104"}

	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			gen := NewGenerator("/tmp/config", protocol)
			masterDockerfile, slaveDockerfile := gen.getProtocolDockerfiles()

			if masterDockerfile == "" || slaveDockerfile == "" {
				t.Error("Dockerfiles should not be empty for valid protocol")
			}

			expectedMaster := "./protocols/" + protocol + "/master/Dockerfile.master"
			expectedSlave := "./protocols/" + protocol + "/slave/Dockerfile.slave"

			if masterDockerfile != expectedMaster {
				t.Errorf("Expected master dockerfile '%s', got '%s'", expectedMaster, masterDockerfile)
			}

			if slaveDockerfile != expectedSlave {
				t.Errorf("Expected slave dockerfile '%s', got '%s'", expectedSlave, slaveDockerfile)
			}
		})
	}
}

func TestInvalidProtocol(t *testing.T) {
	gen := NewGenerator("/tmp/config", "invalid_protocol")
	masterDockerfile, slaveDockerfile := gen.getProtocolDockerfiles()

	if masterDockerfile != "" || slaveDockerfile != "" {
		t.Error("Expected empty dockerfiles for invalid protocol")
	}
}

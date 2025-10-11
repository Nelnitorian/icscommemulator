package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewGenerator(t *testing.T) {
	scenario := Scenario{
		Protocol: "modbus",
		Nodes: []Node{
			{Role: "master"},
			{Role: "slave"},
		},
	}

	gen := NewGenerator(scenario, "/tmp/test_config")

	if gen.protocol != "modbus" {
		t.Errorf("Expected protocol 'modbus', got '%s'", gen.protocol)
	}

	if gen.configPath != "/tmp/test_config" {
		t.Errorf("Expected configPath '/tmp/test_config', got '%s'", gen.configPath)
	}
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"String number", "42", 42},
		{"String non-number", "abc", "abc"},
		{"Float", 42.5, 42},
		{"Int", 42, 42},
		{"Nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToInt(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCraftMaster_Modbus(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "modbus",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	messages := []Message{
		{
			Timestamp:    "0",
			Recurrent:    true,
			Interval:     5,
			IP:           "192.168.1.20",
			Port:         502,
			SlaveID:      1,
			FunctionCode: "3",
			StartAddress: 0,
			Count:        10,
			Values:       []interface{}{},
		},
	}

	err := gen.CraftMaster(messages, 0)
	if err != nil {
		t.Fatalf("CraftMaster() error = %v", err)
	}

	csvPath := filepath.Join(tmpDir, "masters", "0", "master.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Errorf("Master CSV file was not created at %s", csvPath)
	}

	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "timestamp") {
		t.Error("CSV missing header 'timestamp'")
	}
	if !contains(contentStr, "function_code") {
		t.Error("CSV missing header 'function_code'")
	}
}

func TestCraftMaster_DNP3(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "dnp3",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	messages := []Message{
		{
			Timestamp:     "0",
			Recurrent:     true,
			Interval:      5,
			IP:            "192.168.2.20",
			Port:          20000,
			OperationType: "read",
			Group:         30,
			Variation:     5,
			Index:         0,
			MasterID:      2,
			OutstationID:  1,
		},
	}

	err := gen.CraftMaster(messages, 0)
	if err != nil {
		t.Fatalf("CraftMaster() error = %v", err)
	}

	csvPath := filepath.Join(tmpDir, "masters", "0", "master.csv")
	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "operation_type") {
		t.Error("DNP3 CSV missing header 'operation_type'")
	}
	if !contains(contentStr, "outstation_id") {
		t.Error("DNP3 CSV missing header 'outstation_id'")
	}
}

func TestCraftMaster_IEC104(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "iec104",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	messages := []Message{
		{
			Timestamp:     "0",
			Recurrent:     true,
			Interval:      5,
			IP:            "192.168.3.20",
			Port:          2404,
			TypeID:        45,
			CommonAddress: 1,
			IOA:           1001,
			COT:           6,
			Value:         "1",
		},
	}

	err := gen.CraftMaster(messages, 0)
	if err != nil {
		t.Fatalf("CraftMaster() error = %v", err)
	}

	csvPath := filepath.Join(tmpDir, "masters", "0", "master.csv")
	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "type_id") {
		t.Error("IEC104 CSV missing header 'type_id'")
	}
	if !contains(contentStr, "common_address") {
		t.Error("IEC104 CSV missing header 'common_address'")
	}
}

func TestCraftMaster_EmptyMessages(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "modbus",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	err := gen.CraftMaster([]Message{}, 0)
	if err != nil {
		t.Fatalf("CraftMaster() with empty messages error = %v", err)
	}

	csvPath := filepath.Join(tmpDir, "masters", "0", "master.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Errorf("Empty master CSV file was not created")
	}
}

func TestCraftSlave_Modbus(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "modbus",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	slave := Node{
		Role:    "slave",
		IP:      "192.168.1.20",
		Port:    502,
		SlaveID: 1,
		HoldingRegisters: map[string]interface{}{
			"type":   "sequential",
			"values": "1,2,3,4,5",
		},
	}

	err := gen.CraftSlave(slave, 0)
	if err != nil {
		t.Fatalf("CraftSlave() error = %v", err)
	}

	yamlPath := filepath.Join(tmpDir, "slaves", "0", "slave.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Errorf("Slave YAML file was not created at %s", yamlPath)
	}

	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	var result map[string]interface{}
	err = yaml.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Verify unwanted fields are removed
	unwantedFields := []string{"comment", "label", "role", "name", "id"}
	for _, field := range unwantedFields {
		if _, exists := result[field]; exists {
			t.Errorf("Unwanted field '%s' found in slave YAML", field)
		}
	}

	// Verify expected fields are present
	if _, exists := result["ip"]; !exists {
		t.Error("Expected field 'ip' not found in slave YAML")
	}
}

func TestCraftSlave_DNP3(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "dnp3",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	slave := Node{
		Role:         "slave",
		IP:           "192.168.2.20",
		Port:         20000,
		OutstationID: 1,
		MasterID:     2,
		AnalogInputs: map[string]interface{}{
			"count": 20,
			"initial_values": []map[string]interface{}{
				{"index": 0, "value": 100.0},
			},
		},
	}

	err := gen.CraftSlave(slave, 0)
	if err != nil {
		t.Fatalf("CraftSlave() DNP3 error = %v", err)
	}

	yamlPath := filepath.Join(tmpDir, "slaves", "0", "slave.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	var result map[string]interface{}
	err = yaml.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if _, exists := result["outstation_id"]; !exists {
		t.Error("Expected DNP3 field 'outstation_id' not found")
	}
	if _, exists := result["analog_inputs"]; !exists {
		t.Error("Expected DNP3 field 'analog_inputs' not found")
	}
}

func TestCraftSlave_IEC104(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "iec104",
		Nodes:    []Node{},
	}

	gen := NewGenerator(scenario, tmpDir)

	slave := Node{
		Role:          "slave",
		IP:            "192.168.3.20",
		Port:          2404,
		CommonAddress: 1,
		T1:            15,
		T2:            10,
		T3:            20,
		SinglePoints: map[string]interface{}{
			"type":      "sequential",
			"start_ioa": 1001,
			"values":    []int{1, 0, 1, 0},
			"type_id":   "M_SP_NA_1",
		},
	}

	err := gen.CraftSlave(slave, 0)
	if err != nil {
		t.Fatalf("CraftSlave() IEC104 error = %v", err)
	}

	yamlPath := filepath.Join(tmpDir, "slaves", "0", "slave.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	var result map[string]interface{}
	err = yaml.Unmarshal(content, &result)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if _, exists := result["common_address"]; !exists {
		t.Error("Expected IEC104 field 'common_address' not found")
	}
	if _, exists := result["single_points"]; !exists {
		t.Error("Expected IEC104 field 'single_points' not found")
	}
}

func TestGenerate_FullScenario(t *testing.T) {
	tmpDir := t.TempDir()

	scenario := Scenario{
		Protocol: "modbus",
		Nodes: []Node{
			{
				Role: "master",
				Messages: []Message{
					{
						Timestamp:    "0",
						Recurrent:    true,
						Interval:     5,
						IP:           "192.168.1.20",
						Port:         502,
						SlaveID:      1,
						FunctionCode: "3",
						StartAddress: 0,
						Count:        10,
					},
				},
			},
			{
				Role:    "slave",
				IP:      "192.168.1.20",
				Port:    502,
				SlaveID: 1,
				HoldingRegisters: map[string]interface{}{
					"type":   "sequential",
					"values": "1,2,3",
				},
			},
		},
	}

	gen := NewGenerator(scenario, tmpDir)

	err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify master config was created
	masterCSV := filepath.Join(tmpDir, "masters", "0", "master.csv")
	if _, err := os.Stat(masterCSV); os.IsNotExist(err) {
		t.Error("Master CSV was not created")
	}

	// Verify slave config was created
	slaveYAML := filepath.Join(tmpDir, "slaves", "0", "slave.yaml")
	if _, err := os.Stat(slaveYAML); os.IsNotExist(err) {
		t.Error("Slave YAML was not created")
	}
}

func TestClean(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config")

	// Create some files
	err := os.MkdirAll(filepath.Join(configPath, "masters", "0"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFile := filepath.Join(configPath, "masters", "0", "test.csv")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scenario := Scenario{Protocol: "modbus"}
	gen := NewGenerator(scenario, configPath)

	err = gen.Clean()
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Clean() did not remove config directory")
	}
}

func TestIsMaster(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want bool
	}{
		{"Master node", Node{Role: "master"}, true},
		{"Slave node", Node{Role: "slave"}, false},
		{"Empty role", Node{Role: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMaster(tt.node); got != tt.want {
				t.Errorf("IsMaster() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSlave(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want bool
	}{
		{"Slave node", Node{Role: "slave"}, true},
		{"Master node", Node{Role: "master"}, false},
		{"Empty role", Node{Role: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSlave(tt.node); got != tt.want {
				t.Errorf("IsSlave() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMasterCount(t *testing.T) {
	scenario := Scenario{
		Nodes: []Node{
			{Role: "master"},
			{Role: "master"},
			{Role: "slave"},
		},
	}

	gen := NewGenerator(scenario, "/tmp")
	count := gen.GetMasterCount()

	if count != 2 {
		t.Errorf("GetMasterCount() = %d, want 2", count)
	}
}

func TestGetSlaveCount(t *testing.T) {
	scenario := Scenario{
		Nodes: []Node{
			{Role: "master"},
			{Role: "slave"},
			{Role: "slave"},
		},
	}

	gen := NewGenerator(scenario, "/tmp")
	count := gen.GetSlaveCount()

	if count != 2 {
		t.Errorf("GetSlaveCount() = %d, want 2", count)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) >= len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewGenerator(t *testing.T) {
	scenario := Scenario{
		Nodes: []Node{
			{Role: "master"},
			{Role: "slave"},
		},
	}
	configPath := "/tmp/test"

	generator := NewGenerator(scenario, configPath)
	
	assert.Equal(t, scenario, generator.scenario)
	assert.Equal(t, configPath, generator.configPath)
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected interface{}
	}{
		{"123", 123},
		{"abc", "abc"},
		{456, 456},
		{78.9, 78},
		{nil, nil},
	}

	for _, tt := range tests {
		result := ConvertToInt(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCraftMaster(t *testing.T) {
	tempDir := t.TempDir()
	
	generator := NewGenerator(Scenario{}, tempDir)
	
	messages := []Message{
		{
			Timestamp:    "1",
			Recurrent:    true,
			Interval:     5,
			IP:           "192.168.1.10",
			Port:         502,
			SlaveID:      1,
			FunctionCode: "3",
			StartAddress: 0,
			Count:        10,
		},
	}

	err := generator.CraftMaster(messages, 0)
	require.NoError(t, err)

	// Verify CSV file was created
	csvPath := filepath.Join(tempDir, "masters", "0", "master.csv")
	assert.FileExists(t, csvPath)

	// Read and verify content
	content, err := os.ReadFile(csvPath)
	require.NoError(t, err)
	
	csvContent := string(content)
	assert.Contains(t, csvContent, "timestamp")
	assert.Contains(t, csvContent, "192.168.1.10")
	assert.Contains(t, csvContent, "502")
}

func TestCraftSlave(t *testing.T) {
	tempDir := t.TempDir()
	
	generator := NewGenerator(Scenario{}, tempDir)
	
	slave := Node{
		Role:    "slave",
		ID:      "slave1",
		Name:    "Test Slave",
		IP:      "192.168.1.11",
		Port:    502,
		SlaveID: 1,
		HoldingRegisters: map[string]interface{}{
			"type":   "sequential",
			"values": []int{1, 2, 3, 4},
		},
		Identity: map[string]interface{}{
			"vendor_name": "Test Vendor",
			"model_name":  "Test Model",
		},
	}

	err := generator.CraftSlave(slave, 0)
	require.NoError(t, err)

	// Verify YAML file was created
	yamlPath := filepath.Join(tempDir, "slaves", "0", "slave.yaml")
	assert.FileExists(t, yamlPath)

	// Read and verify content
	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	
	var result map[string]interface{}
	err = yaml.Unmarshal(content, &result)
	require.NoError(t, err)
	
	assert.Equal(t, "192.168.1.11", result["ip"])
	assert.Equal(t, 502, result["port"])
	assert.NotContains(t, result, "role") // Should be removed
	assert.NotContains(t, result, "id")   // Should be removed
}

func TestGenerate(t *testing.T) {
	tempDir := t.TempDir()
	
	scenario := Scenario{
		Nodes: []Node{
			{
				Role: "master",
				Messages: []Message{
					{
						Timestamp:    "1",
						IP:           "192.168.1.10",
						Port:         502,
						SlaveID:      1,
						FunctionCode: "3",
					},
				},
			},
			{
				Role:    "slave",
				IP:      "192.168.1.11",
				Port:    502,
				SlaveID: 1,
			},
		},
	}
	
	generator := NewGenerator(scenario, tempDir)
	err := generator.Generate()
	require.NoError(t, err)

	// Verify master files
	masterCSV := filepath.Join(tempDir, "masters", "0", "master.csv")
	assert.FileExists(t, masterCSV)

	// Verify slave files
	slaveYAML := filepath.Join(tempDir, "slaves", "0", "slave.yaml")
	assert.FileExists(t, slaveYAML)
}

func TestFilterFunctions(t *testing.T) {
	nodes := []Node{
		{Role: "master"},
		{Role: "slave"},
		{Role: "master"},
	}

	masters := FilterMasters(nodes)
	slaves := FilterSlaves(nodes)

	assert.Len(t, masters, 2)
	assert.Len(t, slaves, 1)

	for _, master := range masters {
		assert.True(t, IsMaster(master))
		assert.False(t, IsSlave(master))
	}

	for _, slave := range slaves {
		assert.True(t, IsSlave(slave))
		assert.False(t, IsMaster(slave))
	}
}

func TestGenerateFromMap(t *testing.T) {
	tempDir := t.TempDir()
	
	scenarioMap := map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"role": "master",
				"messages": []interface{}{
					map[string]interface{}{
						"timestamp":     "1",
						"ip":           "192.168.1.10",
						"port":         502,
						"slave_id":     1,
						"function_code": "3",
					},
				},
			},
			map[string]interface{}{
				"role":     "slave",
				"ip":       "192.168.1.11",
				"port":     502,
				"slave_id": 1,
			},
		},
	}

	err := GenerateFromMap(scenarioMap, tempDir)
	require.NoError(t, err)

	// Verify files were created
	masterCSV := filepath.Join(tempDir, "masters", "0", "master.csv")
	slaveYAML := filepath.Join(tempDir, "slaves", "0", "slave.yaml")
	
	assert.FileExists(t, masterCSV)
	assert.FileExists(t, slaveYAML)
}

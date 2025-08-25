package scenario

import (
	"os"
	"path/filepath"
	"testing"

	"icscommemulator/pkg/adapter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveScenario(t *testing.T) {
	// Remover la variable no utilizada
	scenarioData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master1", Role: "master", IP: "192.168.1.10"}},
		},
		Edges: []adapter.Edge{},
	}

	err := SaveScenario("test_scenario", scenarioData)
	require.NoError(t, err)

	// Verify files were created
	scenarioDir := filepath.Join(SCENARIO_ROOT_FOLDER, "test_scenario")
	jsonFile := filepath.Join(scenarioDir, "config.json")
	yamlFile := filepath.Join(scenarioDir, "config.yaml")

	assert.DirExists(t, scenarioDir)
	assert.FileExists(t, jsonFile)
	assert.FileExists(t, yamlFile)

	// Cleanup
	os.RemoveAll(scenarioDir)
}

func TestGetCreatedScenarios(t *testing.T) {
	// Create test scenarios
	testScenarios := []string{"scenario1", "scenario2", "scenario3"}
	
	for _, name := range testScenarios {
		scenarioData := adapter.CytoscapeData{
			Protocol:  "modbus",
			IPNetwork: "192.168.1.0/24",
			Nodes:     []adapter.Node{},
			Edges:     []adapter.Edge{},
		}
		err := SaveScenario(name, scenarioData)
		require.NoError(t, err)
	}

	scenarios, err := GetCreatedScenarios()
	require.NoError(t, err)

	// Verify all test scenarios are returned
	for _, testScenario := range testScenarios {
		assert.Contains(t, scenarios, testScenario)
	}

	// Cleanup
	for _, name := range testScenarios {
		os.RemoveAll(filepath.Join(SCENARIO_ROOT_FOLDER, name))
	}
}

func TestCheckScenarioExists(t *testing.T) {
	scenarioName := "test_exists"
	
	// Initially should not exist
	assert.False(t, CheckScenarioExists(scenarioName))

	// Create scenario
	scenarioData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes:     []adapter.Node{},
		Edges:     []adapter.Edge{},
	}
	err := SaveScenario(scenarioName, scenarioData)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, CheckScenarioExists(scenarioName))

	// Cleanup
	os.RemoveAll(filepath.Join(SCENARIO_ROOT_FOLDER, scenarioName))
}

func TestGetCytoscapeScenario(t *testing.T) {
	scenarioName := "test_get"
	originalData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master1", Role: "master", IP: "192.168.1.10"}},
		},
		Edges: []adapter.Edge{},
	}

	// Save scenario
	err := SaveScenario(scenarioName, originalData)
	require.NoError(t, err)

	// Retrieve scenario
	retrievedData, err := GetCytoscapeScenario(scenarioName)
	require.NoError(t, err)

	assert.Equal(t, originalData.Protocol, retrievedData.Protocol)
	assert.Equal(t, originalData.IPNetwork, retrievedData.IPNetwork)
	assert.Len(t, retrievedData.Nodes, 1)
	assert.Equal(t, "master1", retrievedData.Nodes[0].Data.ID)

	// Cleanup
	os.RemoveAll(filepath.Join(SCENARIO_ROOT_FOLDER, scenarioName))
}

func TestDeleteScenario(t *testing.T) {
	scenarioName := "test_delete"
	
	// Create scenario
	scenarioData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes:     []adapter.Node{},
		Edges:     []adapter.Edge{},
	}
	err := SaveScenario(scenarioName, scenarioData)
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, CheckScenarioExists(scenarioName))

	// Delete it
	err = DeleteScenario(scenarioName)
	require.NoError(t, err)

	// Verify it's gone
	assert.False(t, CheckScenarioExists(scenarioName))
}

func TestGetScenarioSize(t *testing.T) {
	scenarioName := "test_size"
	
	// Create scenario
	scenarioData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master1", Role: "master", IP: "192.168.1.10"}},
		},
		Edges: []adapter.Edge{},
	}
	err := SaveScenario(scenarioName, scenarioData)
	require.NoError(t, err)

	// Get size
	size, err := GetScenarioSize(scenarioName)
	require.NoError(t, err)
	
	assert.Greater(t, size, int64(0))

	// Cleanup
	os.RemoveAll(filepath.Join(SCENARIO_ROOT_FOLDER, scenarioName))
}

func TestValidateScenarioFiles(t *testing.T) {
	scenarioName := "test_validate"
	
	// Create scenario
	scenarioData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes:     []adapter.Node{},
		Edges:     []adapter.Edge{},
	}
	err := SaveScenario(scenarioName, scenarioData)
	require.NoError(t, err)

	// Validate files
	err = ValidateScenarioFiles(scenarioName)
	assert.NoError(t, err)

	// Test with non-existent scenario
	err = ValidateScenarioFiles("non_existent")
	assert.Error(t, err)

	// Cleanup
	os.RemoveAll(filepath.Join(SCENARIO_ROOT_FOLDER, scenarioName))
}

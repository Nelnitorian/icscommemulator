package scenario

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"icscommemulator/pkg/adapter"
)

// Constants
const (
	// SCENARIO_ROOT_FOLDER is the root directory for storing scenarios
	SCENARIO_ROOT_FOLDER = "scenarios"
)

// ScenarioFiles represents the files associated with a scenario
type ScenarioFiles struct {
	Name     string `json:"name"`
	JSONPath string `json:"json_path"`
	YAMLPath string `json:"yaml_path"`
}

// init function to ensure the scenarios directory exists
func init() {
	err := os.MkdirAll(SCENARIO_ROOT_FOLDER, 0755)
	if err != nil {
		// Log error but don't panic during init
		fmt.Printf("Warning: failed to create scenarios directory: %v\n", err)
	}
}

// SaveScenario saves a scenario with both JSON and YAML representations
func SaveScenario(name string, rawData adapter.CytoscapeData) error {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	err := os.MkdirAll(scenarioFolder, 0755)
	if err != nil {
		return fmt.Errorf("failed to create scenario folder: %w", err)
	}

	jsonFile := filepath.Join(scenarioFolder, "config.json")
	yamlFile := filepath.Join(scenarioFolder, "config.yaml")

	// Parse cytoscape data to YAML format
	parsedData, err := adapter.ParseCytoscapeJSON(rawData)
	if err != nil {
		return fmt.Errorf("failed to parse cytoscape JSON: %w", err)
	}

	// Save JSON file
	err = saveJSONFile(jsonFile, rawData)
	if err != nil {
		return fmt.Errorf("failed to save JSON file: %w", err)
	}

	// Save YAML file
	err = saveTextFile(yamlFile, parsedData)
	if err != nil {
		return fmt.Errorf("failed to save YAML file: %w", err)
	}

	return nil
}

// SaveScenarioFromMap saves a scenario from a generic map (for compatibility with dynamic data)
func SaveScenarioFromMap(name string, rawData map[string]interface{}) error {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	err := os.MkdirAll(scenarioFolder, 0755)
	if err != nil {
		return fmt.Errorf("failed to create scenario folder: %w", err)
	}

	jsonFile := filepath.Join(scenarioFolder, "config.json")
	yamlFile := filepath.Join(scenarioFolder, "config.yaml")

	// Convert map to CytoscapeData struct for parsing
	dataBytes, err := json.Marshal(rawData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw data: %w", err)
	}

	var cytoscapeData adapter.CytoscapeData
	err = json.Unmarshal(dataBytes, &cytoscapeData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to CytoscapeData: %w", err)
	}

	// Parse cytoscape data to YAML format
	parsedData, err := adapter.ParseCytoscapeJSON(cytoscapeData)
	if err != nil {
		return fmt.Errorf("failed to parse cytoscape JSON: %w", err)
	}

	// Save JSON file
	err = saveJSONFromMap(jsonFile, rawData)
	if err != nil {
		return fmt.Errorf("failed to save JSON file: %w", err)
	}

	// Save YAML file
	err = saveTextFile(yamlFile, parsedData)
	if err != nil {
		return fmt.Errorf("failed to save YAML file: %w", err)
	}

	return nil
}

// GetCreatedScenarios returns a list of all created scenario names
func GetCreatedScenarios() ([]string, error) {
	entries, err := os.ReadDir(SCENARIO_ROOT_FOLDER)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() {
			scenarios = append(scenarios, entry.Name())
		}
	}

	return scenarios, nil
}

// GetCytoscapeScenario retrieves a scenario in Cytoscape format
func GetCytoscapeScenario(name string) (adapter.CytoscapeData, error) {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	jsonFile := filepath.Join(scenarioFolder, "config.json")

	var data adapter.CytoscapeData

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		return data, fmt.Errorf("failed to read JSON file: %w", err)
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return data, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	return data, nil
}

// GetCytoscapeScenarioAsMap retrieves a scenario as a generic map
func GetCytoscapeScenarioAsMap(name string) (map[string]interface{}, error) {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	jsonFile := filepath.Join(scenarioFolder, "config.json")

	var data map[string]interface{}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	return data, nil
}

// CheckScenarioExists checks if a scenario with the given name exists
func CheckScenarioExists(name string) bool {
	scenarioPath := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	info, err := os.Stat(scenarioPath)
	return err == nil && info.IsDir()
}

// GetPythonScenario retrieves a scenario in the parsed Python format (YAML)
func GetPythonScenario(name string) (map[string]interface{}, error) {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	yamlFile := filepath.Join(scenarioFolder, "config.yaml")

	var data map[string]interface{}

	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	err = yaml.Unmarshal(yamlData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data: %w", err)
	}

	return data, nil
}

// DeleteScenario removes a scenario and all its files
func DeleteScenario(name string) error {
	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)

	if !CheckScenarioExists(name) {
		return fmt.Errorf("scenario '%s' does not exist", name)
	}

	err := os.RemoveAll(scenarioFolder)
	if err != nil {
		return fmt.Errorf("failed to delete scenario folder: %w", err)
	}

	return nil
}

// ListScenariosWithDetails returns detailed information about all scenarios
func ListScenariosWithDetails() ([]ScenarioFiles, error) {
	scenarios, err := GetCreatedScenarios()
	if err != nil {
		return nil, err
	}

	var details []ScenarioFiles
	for _, name := range scenarios {
		scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
		detail := ScenarioFiles{
			Name:     name,
			JSONPath: filepath.Join(scenarioFolder, "config.json"),
			YAMLPath: filepath.Join(scenarioFolder, "config.yaml"),
		}
		details = append(details, detail)
	}

	return details, nil
}

// ValidateScenarioFiles checks if both JSON and YAML files exist for a scenario
func ValidateScenarioFiles(name string) error {
	if !CheckScenarioExists(name) {
		return fmt.Errorf("scenario '%s' does not exist", name)
	}

	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	jsonFile := filepath.Join(scenarioFolder, "config.json")
	yamlFile := filepath.Join(scenarioFolder, "config.yaml")

	if _, err := os.Stat(jsonFile); err != nil {
		return fmt.Errorf("JSON file missing for scenario '%s': %w", name, err)
	}

	if _, err := os.Stat(yamlFile); err != nil {
		return fmt.Errorf("YAML file missing for scenario '%s': %w", name, err)
	}

	return nil
}

// GetScenarioSize returns the total size of a scenario's files
func GetScenarioSize(name string) (int64, error) {
	if !CheckScenarioExists(name) {
		return 0, fmt.Errorf("scenario '%s' does not exist", name)
	}

	scenarioFolder := filepath.Join(SCENARIO_ROOT_FOLDER, name)
	var totalSize int64

	err := filepath.WalkDir(scenarioFolder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate scenario size: %w", err)
	}

	return totalSize, nil
}

// Helper functions

// saveJSONFile saves structured data as JSON
func saveJSONFile(filename string, data adapter.CytoscapeData) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	return encoder.Encode(data)
}

// saveJSONFromMap saves map data as JSON
func saveJSONFromMap(filename string, data map[string]interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	return encoder.Encode(data)
}

// saveTextFile saves text content to a file
func saveTextFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

// GetScenariosRoot returns the root directory for scenarios
func GetScenariosRoot() string {
	return SCENARIO_ROOT_FOLDER
}

// SetScenariosRoot allows changing the root directory for scenarios (useful for testing)
func SetScenariosRoot(path string) error {
	// This is a bit tricky since we can't modify constants at runtime
	// For production use, you might want to use a global variable instead
	return fmt.Errorf("changing scenarios root at runtime not implemented - modify SCENARIO_ROOT_FOLDER constant")
}

// Example usage and utility functions

// ExampleUsage demonstrates how to use the scenario handler
func ExampleUsage() error {
	// Create example scenario data
	exampleData := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{
				Data: adapter.NodeData{
					ID:   "master_1",
					Role: "master",
					Name: "Master Node",
					IP:   "192.168.1.10",
				},
			},
			{
				Data: adapter.NodeData{
					ID:   "slave_1",
					Role: "slave",
					Name: "Slave Node",
					IP:   "192.168.1.11",
				},
			},
		},
		Edges: []adapter.Edge{},
	}

	// Save the scenario
	err := SaveScenario("example_scenario", exampleData)
	if err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}

	// List all scenarios
	scenarios, err := GetCreatedScenarios()
	if err != nil {
		return fmt.Errorf("failed to get scenarios: %w", err)
	}

	fmt.Printf("Available scenarios: %v\n", scenarios)

	// Check if scenario exists
	exists := CheckScenarioExists("example_scenario")
	fmt.Printf("Example scenario exists: %t\n", exists)

	// Get scenario back
	retrievedData, err := GetCytoscapeScenario("example_scenario")
	if err != nil {
		return fmt.Errorf("failed to retrieve scenario: %w", err)
	}

	fmt.Printf("Retrieved scenario protocol: %s\n", retrievedData.Protocol)

	return nil
}

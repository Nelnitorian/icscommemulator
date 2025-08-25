package config

import (
	"encoding/csv"
	"fmt"
	"icscommemulator/pkg/logger"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Message represents a message configuration for master nodes
type Message struct {
	Timestamp    string        `json:"timestamp" yaml:"timestamp"`
	Recurrent    bool          `json:"recurrent" yaml:"recurrent"`
	Interval     int           `json:"interval,omitempty" yaml:"interval,omitempty"`
	IP           string        `json:"ip" yaml:"ip"`
	Port         int           `json:"port" yaml:"port"`
	SlaveID      int           `json:"slave_id" yaml:"slave_id"`
	FunctionCode string        `json:"function_code" yaml:"function_code"`
	StartAddress int           `json:"start_address" yaml:"start_address"`
	Count        interface{}   `json:"count,omitempty" yaml:"count,omitempty"`
	Values       []interface{} `json:"values,omitempty" yaml:"values,omitempty"`
}

// Node represents a node in the scenario
type Node struct {
	Role     string    `json:"role" yaml:"role"`
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`
	
	// Additional fields that might be present
	ID               string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name             string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Label            string                 `json:"label,omitempty" yaml:"label,omitempty"`
	Comment          string                 `json:"comment,omitempty" yaml:"comment,omitempty"`
	IP               string                 `json:"ip,omitempty" yaml:"ip,omitempty"`
	Port             interface{}            `json:"port,omitempty" yaml:"port,omitempty"`
	SlaveID          interface{}            `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	HoldingRegisters map[string]interface{} `json:"holding_registers,omitempty" yaml:"holding_registers,omitempty"`
	Coils            map[string]interface{} `json:"coils,omitempty" yaml:"coils,omitempty"`
	DiscreteInputs   map[string]interface{} `json:"discrete_inputs,omitempty" yaml:"discrete_inputs,omitempty"`
	InputRegisters   map[string]interface{} `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`
	Identity         map[string]interface{} `json:"identity,omitempty" yaml:"identity,omitempty"`
}

// Scenario represents the complete scenario configuration
type Scenario struct {
	Nodes []Node `json:"nodes" yaml:"nodes"`
}

// Generator manages scenario configuration file generation
type Generator struct {
	scenario   Scenario
	configPath string
}

// NewGenerator creates a new scenario configuration generator
func NewGenerator(scenario Scenario, configPath string) *Generator {
	return &Generator{
		scenario:   scenario,
		configPath: configPath,
	}
}

// ConvertToInt attempts to convert a value to an integer, returns the original value if conversion fails
func ConvertToInt(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		return v
	case float64:
		return int(v)
	case int:
		return v
	default:
		return v
	}
}

// CraftMaster creates configuration files for master nodes
func (g *Generator) CraftMaster(messages []Message, index int) error {
	masterDir := filepath.Join(g.configPath, "masters", strconv.Itoa(index))
	err := os.MkdirAll(masterDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create master directory: %w", err)
	}

	csvFile := filepath.Join(masterDir, "master.csv")
	
	if len(messages) == 0 {
		// Create empty file if no messages
		file, err := os.Create(csvFile)
		if err != nil {
			return fmt.Errorf("failed to create empty CSV file: %w", err)
		}
		file.Close()
		return nil
	}

	// Create CSV file
	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{
		"timestamp", "recurrent", "interval", "ip", "port", "slave_id",
		"function_code", "start_address", "count", "values",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, message := range messages {
		// Convert count field (equivalent to pandas logic)
		count := ConvertToInt(message.Count)
		countStr := ""
		if count != nil {
			switch c := count.(type) {
			case int:
				countStr = strconv.Itoa(c)
			case string:
				countStr = c
			}
		}

		// Convert values to string representation
		valuesStr := ""
		if len(message.Values) > 0 {
			// Simple string representation - you might want to make this more sophisticated
			valuesStr = fmt.Sprintf("%v", message.Values)
		}

		row := []string{
			message.Timestamp,
			strconv.FormatBool(message.Recurrent),
			strconv.Itoa(message.Interval),
			message.IP,
			strconv.Itoa(message.Port),
			strconv.Itoa(message.SlaveID),
			message.FunctionCode,
			strconv.Itoa(message.StartAddress),
			countStr,
			valuesStr,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// CraftSlave creates configuration files for slave nodes
func (g *Generator) CraftSlave(slave Node, index int) error {
	slaveDir := filepath.Join(g.configPath, "slaves", strconv.Itoa(index))
	
	logger.Debug("Creating slave directory: %s", slaveDir)
	err := os.MkdirAll(slaveDir, 0755)
	if err != nil {
		logger.Error("Failed to create slave directory %s: %v", slaveDir, err)
		return fmt.Errorf("failed to create slave directory: %w", err)
	}
	
	logger.Debug("Slave directory created successfully: %s", slaveDir)

	// Create a copy of the slave node and remove unwanted fields
	slaveCopy := make(map[string]interface{})
	
	// Convert struct to map for easier manipulation
	slaveData, err := yaml.Marshal(slave)
	if err != nil {
		logger.Error("Failed to marshal slave data: %v", err)
		return fmt.Errorf("failed to marshal slave data: %w", err)
	}
	
	logger.Debug("Slave data marshaled: %s", string(slaveData))
	
	err = yaml.Unmarshal(slaveData, &slaveCopy)
	if err != nil {
		logger.Error("Failed to unmarshal slave data: %v", err)
		return fmt.Errorf("failed to unmarshal slave data: %w", err)
	}

	// Remove unwanted keys
	fieldsToRemove := []string{"comment", "label", "role", "name", "id"}
	for _, key := range fieldsToRemove {
		delete(slaveCopy, key)
	}
	
	logger.DebugStruct("Cleaned slave data", slaveCopy)

	// Write YAML file
	yamlFile := filepath.Join(slaveDir, "slave.yaml")
	logger.Debug("Creating slave YAML file: %s", yamlFile)
	
	file, err := os.Create(yamlFile)
	if err != nil {
		logger.Error("Failed to create YAML file %s: %v", yamlFile, err)
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	
	err = encoder.Encode(slaveCopy)
	if err != nil {
		logger.Error("Failed to encode YAML to file %s: %v", yamlFile, err)
		return fmt.Errorf("failed to encode YAML: %v", err)
	}

	// Verify file was created and is actually a file
	fileInfo, err := os.Stat(yamlFile)
	if err != nil {
		logger.Error("Failed to stat created file %s: %v", yamlFile, err)
		return fmt.Errorf("failed to verify created file: %w", err)
	}
	
	if fileInfo.IsDir() {
		logger.Error("Created path is a directory instead of file: %s", yamlFile)
		return fmt.Errorf("created path is a directory instead of file: %s", yamlFile)
	}
	
	logger.Info("Successfully created slave config file: %s (size: %d bytes)", yamlFile, fileInfo.Size())
	return nil
}

// Clean removes the configuration directory and all its contents
func (g *Generator) Clean() error {
	if _, err := os.Stat(g.configPath); err == nil {
		return os.RemoveAll(g.configPath)
	}
	return nil
}

// Generate creates the configuration files for the scenario
func (g *Generator) Generate() error {
	logger.Info("Starting configuration generation in: %s", g.configPath)
	
	// Clean existing configuration
	err := g.Clean()
	if err != nil {
		logger.Error("Failed to clean configuration path %s: %v", g.configPath, err)
		return fmt.Errorf("failed to clean configuration path: %w", err)
	}
	
	logger.Debug("Configuration path cleaned successfully")

	// Generate master configurations
	masterIndex := 0
	for _, node := range g.scenario.Nodes {
		if IsMaster(node) {
			logger.Debug("Generating master config %d", masterIndex)
			err := g.CraftMaster(node.Messages, masterIndex)
			if err != nil {
				logger.Error("Failed to craft master %d: %v", masterIndex, err)
				return fmt.Errorf("failed to craft master %d: %w", masterIndex, err)
			}
			logger.Info("Successfully generated master config %d", masterIndex)
			masterIndex++
		}
	}

	// Generate slave configurations
	slaveIndex := 0
	for _, node := range g.scenario.Nodes {
		if IsSlave(node) {
			logger.Debug("Generating slave config %d", slaveIndex)
			err := g.CraftSlave(node, slaveIndex)
			if err != nil {
				logger.Error("Failed to craft slave %d: %v", slaveIndex, err)
				return fmt.Errorf("failed to craft slave %d: %w", slaveIndex, err)
			}
			logger.Info("Successfully generated slave config %d", slaveIndex)
			slaveIndex++
		}
	}

	logger.Info("Configuration generation completed successfully")
	return nil
}
// Helper functions

// IsMaster checks if a node is a master node
func IsMaster(node Node) bool {
	return node.Role == "master"
}

// IsSlave checks if a node is a slave node
func IsSlave(node Node) bool {
	return node.Role == "slave"
}

// FilterMasters returns only the master nodes from a slice of nodes
func FilterMasters(nodes []Node) []Node {
	var masters []Node
	for _, node := range nodes {
		if IsMaster(node) {
			masters = append(masters, node)
		}
	}
	return masters
}

// FilterSlaves returns only the slave nodes from a slice of nodes
func FilterSlaves(nodes []Node) []Node {
	var slaves []Node
	for _, node := range nodes {
		if IsSlave(node) {
			slaves = append(slaves, node)
		}
	}
	return slaves
}

// GetMasterCount returns the number of master nodes
func (g *Generator) GetMasterCount() int {
	count := 0
	for _, node := range g.scenario.Nodes {
		if IsMaster(node) {
			count++
		}
	}
	return count
}

// GetSlaveCount returns the number of slave nodes
func (g *Generator) GetSlaveCount() int {
	count := 0
	for _, node := range g.scenario.Nodes {
		if IsSlave(node) {
			count++
		}
	}
	return count
}

// GenerateFromMap creates a generator from a map[string]interface{} (for JSON compatibility)
func GenerateFromMap(scenarioMap map[string]interface{}, configPath string) error {
	// Convert map to Scenario struct
	scenarioData, err := yaml.Marshal(scenarioMap)
	if err != nil {
		return fmt.Errorf("failed to marshal scenario map: %w", err)
	}

	var scenario Scenario
	err = yaml.Unmarshal(scenarioData, &scenario)
	if err != nil {
		return fmt.Errorf("failed to unmarshal scenario: %w", err)
	}

	generator := NewGenerator(scenario, configPath)
	return generator.Generate()
}

// Example usage function
func ExampleUsage() error {
	// Create example scenario
	scenario := Scenario{
		Nodes: []Node{
			{
				Role: "master",
				Messages: []Message{
					{
						Timestamp:    "2023-01-01T10:00:00Z",
						Recurrent:    true,
						Interval:     5,
						IP:           "192.168.1.10",
						Port:         502,
						SlaveID:      1,
						FunctionCode: "03",
						StartAddress: 0,
						Count:        10,
						Values:       []interface{}{1, 2, 3},
					},
				},
			},
			{
				Role:    "slave",
				IP:      "192.168.1.11",
				Port:    502,
				SlaveID: 1,
				HoldingRegisters: map[string]interface{}{
					"type":   "sequential",
					"values": "1,2,3,4,5",
				},
				Identity: map[string]interface{}{
					"vendor_name": "Example Vendor",
					"model_name":  "Example Model",
				},
			},
		},
	}

	generator := NewGenerator(scenario, "/tmp/ICSCommEmulator")
	return generator.Generate()
}

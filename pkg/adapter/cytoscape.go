package adapter

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Level int

const (
	ERROR Level = iota + 1
	WARNING
)

type Protocol string

const (
	MODBUS Protocol = "modbus"
)

type NodeData struct {
	ID               string                 `json:"id" yaml:"id"`
	Label            string                 `json:"label,omitempty" yaml:"label,omitempty"`
	Role             string                 `json:"role" yaml:"role"`
	Name             string                 `json:"name" yaml:"name"`
	IP               string                 `json:"ip" yaml:"ip"`
	Port             interface{}            `json:"port,omitempty" yaml:"port,omitempty"`
	SlaveID          interface{}            `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	Comment          string                 `json:"comment,omitempty" yaml:"comment,omitempty"`
	Messages         []Message              `json:"messages,omitempty" yaml:"messages,omitempty"`
	HoldingRegisters RegisterConfig         `json:"holding_registers,omitempty" yaml:"holding_registers,omitempty"`
	Coils            RegisterConfig         `json:"coils,omitempty" yaml:"coils,omitempty"`
	DiscreteInputs   RegisterConfig         `json:"discrete_inputs,omitempty" yaml:"discrete_inputs,omitempty"`
	InputRegisters   RegisterConfig         `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`
	Identity         map[string]interface{} `json:"identity,omitempty" yaml:"identity,omitempty"`
	Position         *Position              `json:"position,omitempty" yaml:"-"`
	Group            string                 `json:"group,omitempty" yaml:"-"`
	Removed          bool                   `json:"removed,omitempty" yaml:"-"`
	Selected         bool                   `json:"selected,omitempty" yaml:"-"`
	Selectable       bool                   `json:"selectable,omitempty" yaml:"-"`
	Locked           bool                   `json:"locked,omitempty" yaml:"-"`
	Grabbable        bool                   `json:"grabbable,omitempty" yaml:"-"`
	Pannable         bool                   `json:"pannable,omitempty" yaml:"-"`
	Classes          string                 `json:"classes,omitempty" yaml:"-"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type RegisterConfig struct {
	Type   string      `json:"type,omitempty" yaml:"type,omitempty"`
	Values interface{} `json:"values,omitempty" yaml:"values,omitempty"`
}

type Node struct {
	Data    NodeData `json:"data" yaml:"data"`
	Classes string   `json:"classes,omitempty" yaml:"classes,omitempty"`
}

type EdgeData struct {
	ID           string    `json:"id" yaml:"id"`
	Source       string    `json:"source" yaml:"source"`
	Target       string    `json:"target" yaml:"target"`
	Messages     []Message `json:"messages,omitempty" yaml:"messages,omitempty"`
	Timestamp    string    `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	FunctionCode string    `json:"function_code,omitempty" yaml:"function_code,omitempty"`
	StartAddress int       `json:"start_address,omitempty" yaml:"start_address,omitempty"`
	InputRegisters string  `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`
}

type Edge struct {
	Data EdgeData `json:"data" yaml:"data"`
}

type Message struct {
	Timestamp    int        `json:"timestamp" yaml:"timestamp"`
	Recurrent    bool          `json:"recurrent" yaml:"recurrent"`
	Interval     *int           `json:"interval,omitempty" yaml:"interval,omitempty"`
	IP           string        `json:"ip" yaml:"ip"`
	Port         int           `json:"port" yaml:"port"`
	SlaveID      int           `json:"slave_id" yaml:"slave_id"`
	FunctionCode int        `json:"function_code" yaml:"function_code"`
	StartAddress int           `json:"start_address" yaml:"start_address"`
	Count        int           `json:"count,omitempty" yaml:"count,omitempty"`
	Values       []interface{} `json:"values,omitempty" yaml:"values,omitempty"`
}

type CytoscapeData struct {
	Protocol  string `json:"protocol" yaml:"protocol"`
	IPNetwork string `json:"ip_network" yaml:"ip_network"`
	Nodes     []Node `json:"nodes" yaml:"nodes"`
	Edges     []Edge `json:"edges" yaml:"edges"`
}

type YAMLData struct {
	Protocol  string     `yaml:"protocol"`
	IPNetwork string     `yaml:"ip_network"`
	Nodes     []NodeData `yaml:"nodes"`
}

// ValidateCytoscapeScenario validates a cytoscape scenario based on the given level
func ValidateCytoscapeScenario(data CytoscapeData, level Level) []string {
	var logs []string

	if level >= ERROR {
		// Check if all IDs are different
		nodeIDs := make([]string, len(data.Nodes))
		idCount := make(map[string]int)
		
		for i, node := range data.Nodes {
			nodeIDs[i] = node.Data.ID
			idCount[node.Data.ID]++
		}

		var duplicates []string
		for id, count := range idCount {
			if count > 1 {
				duplicates = append(duplicates, id)
			}
		}

		if len(duplicates) > 0 {
			logs = append(logs, fmt.Sprintf("[ERROR] All node IDs are not unique. Duplicates: %v", duplicates))
		}

		// Check if edges connect nodes with different roles
		nodeRoles := make(map[string]string)
		for _, node := range data.Nodes {
			nodeRoles[node.Data.ID] = node.Data.Role
		}

		for _, edge := range data.Edges {
			sourceRole := nodeRoles[edge.Data.Source]
			targetRole := nodeRoles[edge.Data.Target]
			if sourceRole == targetRole {
				logs = append(logs, fmt.Sprintf("[ERROR] Edge %s connects nodes with the same role (%s)", edge.Data.ID, sourceRole))
			}
		}

		// Check if IPs are within range
		_, ipNet, err := net.ParseCIDR(data.IPNetwork)
		if err != nil {
			logs = append(logs, fmt.Sprintf("[ERROR] Invalid IP network: %s", data.IPNetwork))
		} else {
			for _, node := range data.Nodes {
				ip := net.ParseIP(node.Data.IP)
				if ip == nil {
					logs = append(logs, fmt.Sprintf("[ERROR] Invalid IP %s for node %s", node.Data.IP, node.Data.ID))
				} else if !ipNet.Contains(ip) {
					logs = append(logs, fmt.Sprintf("[ERROR] IP %s of node %s is out of range", node.Data.IP, node.Data.ID))
				}
			}
		}
	}

	if level >= WARNING {
		// Check if there's a slave without any defined register
		for _, node := range data.Nodes {
			if node.Data.Role == "slave" {
				hasRegisters := false
				if hasValues(node.Data.HoldingRegisters.Values) ||
				   hasValues(node.Data.Coils.Values) ||
				   hasValues(node.Data.DiscreteInputs.Values) ||
				   hasValues(node.Data.InputRegisters.Values) {
					hasRegisters = true
				}
				
				if !hasRegisters {
					logs = append(logs, fmt.Sprintf("[WARNING] Slave node %s has no defined registers", node.Data.ID))
				}
			}
		}

		// Check edges with empty message fields
		for _, edge := range data.Edges {
			isEmpty := edge.Data.Timestamp == "" || 
					  edge.Data.FunctionCode == "" || 
					  edge.Data.StartAddress < 0 || 
					  edge.Data.InputRegisters == ""
					  
			if isEmpty {
				logs = append(logs, fmt.Sprintf("[WARNING] Communication %s → %s has empty values.", edge.Data.Source, edge.Data.Target))
			}
		}

		// Find nodes without links
		linkedNodeIDs := make(map[string]bool)
		for _, edge := range data.Edges {
			linkedNodeIDs[edge.Data.Source] = true
			linkedNodeIDs[edge.Data.Target] = true
		}

		for _, node := range data.Nodes {
			if !linkedNodeIDs[node.Data.ID] {
				logs = append(logs, fmt.Sprintf("[WARNING] Node without link: %s", node.Data.ID))
			}
		}
	}

	return logs
}

func hasValues(values interface{}) bool {
	if values == nil {
		return false
	}
	if str, ok := values.(string); ok {
		return strings.TrimSpace(str) != ""
	}
	return true
}

// CleanDictValues cleans dictionary values by removing illegal characters
func CleanDictValues(inputMap map[string]interface{}) map[string]interface{} {
	allowedPattern := regexp.MustCompile(`[a-zA-Z0-9\s.,/:_-]`)
	cleanedMap := make(map[string]interface{})

	for key, value := range inputMap {
		if str, ok := value.(string); ok {
			matches := allowedPattern.FindAllString(str, -1)
			cleanedMap[key] = strings.Join(matches, "")
		} else {
			cleanedMap[key] = value
		}
	}

	return cleanedMap
}

// ParseCytoscapeJSON parses cytoscape JSON to YAML format
func ParseCytoscapeJSON(data CytoscapeData) (string, error) {
	yamlData := YAMLData{
		Protocol:  data.Protocol,
		IPNetwork: data.IPNetwork,
		Nodes:     []NodeData{},
	}

	// Create messages dictionary for master nodes
	messagesDict := make(map[string][]Message)
	for _, node := range data.Nodes {
		if node.Data.Role == "master" {
			messagesDict[node.Data.ID] = []Message{}
		}
	}

	// Parse edges to generate messages for master nodes
	for _, edge := range data.Edges {
		targetNode := findFirstMatchingNode(data.Nodes, edge.Data.Target)
		if targetNode == nil {
			continue
		}

		for _, message := range edge.Data.Messages {
			// Convert port and slave_id to integers
			port := 502 // default
			if p, ok := targetNode.Data.Port.(string); ok {
				if pInt, err := parseIntFromString(p); err == nil {
					port = pInt
				}
			}

			slaveID := 1 // default
			if s, ok := targetNode.Data.SlaveID.(string); ok {
				if sInt, err := parseIntFromString(s); err == nil {
					slaveID = sInt
				}
			}

			msg := Message{
				Timestamp:    message.Timestamp,
				Recurrent:    message.Recurrent,
				Interval:     message.Interval,
				IP:           targetNode.Data.IP,
				Port:         port,
				SlaveID:      slaveID,
				FunctionCode: message.FunctionCode,
				StartAddress: message.StartAddress,
				Count:        message.Count,
				Values:       message.Values,
			}

			messagesDict[edge.Data.Source] = append(messagesDict[edge.Data.Source], msg)
		}
	}

	// Add nodes to yamlData
	for _, node := range data.Nodes {
		nodeData := node.Data
		
		if nodeData.Role == "master" {
			nodeData.Messages = messagesDict[nodeData.ID]
		}

		// Clean data
		if nodeData.Role == "slave" {
			if port, ok := nodeData.Port.(string); ok {
				if p, err := parseIntFromString(port); err == nil {
					nodeData.Port = p
				}
			}
			if slaveID, ok := nodeData.SlaveID.(string); ok {
				if s, err := parseIntFromString(slaveID); err == nil {
					nodeData.SlaveID = s
				}
			}
			if nodeData.Identity != nil {
				nodeData.Identity = CleanDictValues(nodeData.Identity)
			}
		}

		yamlData.Nodes = append(yamlData.Nodes, nodeData)
	}

	// Convert to YAML
	yamlOutput, err := yaml.Marshal(yamlData)
	if err != nil {
		return "", err
	}

	return string(yamlOutput), nil
}

func findFirstMatchingNode(nodes []Node, nodeID string) *Node {
	for _, node := range nodes {
		if node.Data.ID == nodeID {
			return &node
		}
	}
	return nil
}

func parseIntFromString(s string) (int, error) {
	// Implementation depends on specific parsing logic needed
	// This is a simplified version
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// GenerateNetworkNodes generates master and slave nodes
func GenerateNetworkNodes(proto Protocol, ipBase net.IP, masterNodes, slaveNodes int) ([]Node, []Edge) {
	nodes := []Node{}
	edges := []Edge{}
	
	currentIP := ipBase

	// Create master nodes
	for i := 0; i < masterNodes; i++ {
		masterNode := Node{
			Data: NodeData{
				ID:    fmt.Sprintf("master_%d", i),
				Label: "data(name)",
				Role:  "master",
				Name:  fmt.Sprintf("master_%d", i),
				IP:    currentIP.String(),
			},
			Classes: "master",
		}
		
		nodes = append(nodes, masterNode)
		currentIP = incrementIP(currentIP)
	}

	// Create slave nodes
	for i := 0; i < slaveNodes; i++ {
		data := NodeData{
			ID:    fmt.Sprintf("slave_%d", i),
			Label: "data(name)",
			Role:  "slave",
			Name:  fmt.Sprintf("slave_%d", i),
			IP:    currentIP.String(),
		}

		if proto == MODBUS {
			data.Port = "502"
			data.SlaveID = "1"
			data.Comment = ""
			data.HoldingRegisters = RegisterConfig{Type: "sequential", Values: ""}
			data.Coils = RegisterConfig{Type: "sequential", Values: ""}
			data.DiscreteInputs = RegisterConfig{Type: "sequential", Values: ""}
			data.InputRegisters = RegisterConfig{Type: "sequential", Values: ""}
			data.Identity = map[string]interface{}{
				"major_minor_revision":   "",
				"model_name":            "",
				"product_code":          "",
				"product_name":          "",
				"user_application_name": "",
				"vendor_name":           "",
				"vendor_url":            "",
			}
		}

		slaveNode := Node{
			Data:    data,
			Classes: "slave",
		}

		nodes = append(nodes, slaveNode)
		currentIP = incrementIP(currentIP)
	}

	return nodes, edges
}

func incrementIP(ip net.IP) net.IP {
	// Create a copy of the IP
	result := make(net.IP, len(ip))
	copy(result, ip)
	
	// Increment the IP address
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] > 0 {
			break
		}
	}
	
	return result
}

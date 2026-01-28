package adapter

import (
	"net"
	"strings"
	"testing"
)

func TestValidateCytoscapeScenarioErrors(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []Node{
			{Data: NodeData{ID: "node_1", Role: "master", IP: "192.168.1.10"}},
			{Data: NodeData{ID: "node_1", Role: "slave", IP: "192.168.1.11"}},
		},
		Edges: []Edge{},
	}

	logs := ValidateCytoscapeScenario(data, ERROR)
	if len(logs) == 0 {
		t.Fatalf("expected validation errors for duplicate IDs")
	}
}

func TestParseCytoscapeJSONBasic(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []Node{
			{Data: NodeData{ID: "master_0", Role: "master", Name: "Master", IP: "192.168.1.10"}},
			{Data: NodeData{
				ID:               "slave_0",
				Role:             "slave",
				Name:             "Slave",
				IP:               "192.168.1.11",
				Port:             "502",
				SlaveID:          "1",
				HoldingRegisters: map[string]int{"0": 1},
			}},
		},
		Edges: []Edge{
			{Data: EdgeData{ID: "edge_0", Source: "master_0", Target: "slave_0"}},
		},
	}

	yamlOutput, err := ParseCytoscapeJSON(data)
	if err != nil {
		t.Fatalf("ParseCytoscapeJSON() error = %v", err)
	}

	if !strings.Contains(yamlOutput, "protocol: modbus") {
		t.Fatalf("expected protocol in YAML output")
	}
	if !strings.Contains(yamlOutput, "ip_network: 192.168.1.0/24") {
		t.Fatalf("expected ip_network in YAML output")
	}
	if !strings.Contains(yamlOutput, "master_0") || !strings.Contains(yamlOutput, "slave_0") {
		t.Fatalf("expected node identifiers in YAML output")
	}
}

func TestGenerateNetworkNodesDefaults(t *testing.T) {
	baseIP := net.ParseIP("192.168.5.10")
	nodes, edges := GenerateNetworkNodes(MODBUS, baseIP, 1, 2)

	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(edges))
	}

	slaveCount := 0
	for _, node := range nodes {
		if node.Data.Role == "slave" {
			slaveCount++
			if node.Data.Port != "502" {
				t.Fatalf("expected modbus slave port 502, got %v", node.Data.Port)
			}
			if node.Data.SlaveID != "1" {
				t.Fatalf("expected modbus slave_id 1, got %v", node.Data.SlaveID)
			}
		}
	}

	if slaveCount != 2 {
		t.Fatalf("expected 2 slaves, got %d", slaveCount)
	}
}

func TestCleanDictValues(t *testing.T) {
	input := map[string]interface{}{
		"clean":   "hello world",
		"special": "test@#$%^&*()value",
		"number":  123,
	}

	result := CleanDictValues(input)
	if result["clean"] != "hello world" {
		t.Fatalf("unexpected clean value: %v", result["clean"])
	}
	if result["special"] != "testvalue" {
		t.Fatalf("unexpected special value: %v", result["special"])
	}
	if result["number"] != 123 {
		t.Fatalf("unexpected number value: %v", result["number"])
	}
}

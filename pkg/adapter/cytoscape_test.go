package adapter

import (
	"net"
	"testing"
)

func TestValidateCytoscapeScenario(t *testing.T) {
	tests := []struct {
		name     string
		scenario CytoscapeData
		level    Level
		wantLogs int
	}{
		{
			name: "Valid Modbus scenario",
			scenario: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{
						Data: NodeData{
							ID:   "master_0",
							Role: "master",
							IP:   "192.168.1.10",
						},
					},
					{
						Data: NodeData{
							ID:               "slave_0",
							Role:             "slave",
							IP:               "192.168.1.20",
							Port:             "502",
							SlaveID:          "1",
							HoldingRegisters: RegisterConfig{Type: "sequential", Values: "1,2,3"},
						},
					},
				},
				Edges: []Edge{
					{Data: EdgeData{ID: "edge_0", Source: "master_0", Target: "slave_0"}},
				},
			},
			level:    ERROR,
			wantLogs: 0,
		},
		{
			name: "Valid DNP3 scenario",
			scenario: CytoscapeData{
				Protocol:  "dnp3",
				IPNetwork: "192.168.2.0/24",
				Nodes: []Node{
					{
						Data: NodeData{
							ID:   "master_0",
							Role: "master",
							IP:   "192.168.2.10",
						},
					},
					{
						Data: NodeData{
							ID:           "slave_0",
							Role:         "slave",
							IP:           "192.168.2.20",
							Port:         "20000",
							OutstationID: 1,
							MasterID:     2,
							AnalogInputs: &DNP3PointConfig{
								Count:         10,
								InitialValues: []map[string]interface{}{},
							},
						},
					},
				},
				Edges: []Edge{
					{Data: EdgeData{ID: "edge_0", Source: "master_0", Target: "slave_0"}},
				},
			},
			level:    ERROR,
			wantLogs: 0,
		},
		{
			name: "Valid IEC104 scenario",
			scenario: CytoscapeData{
				Protocol:  "iec104",
				IPNetwork: "192.168.3.0/24",
				Nodes: []Node{
					{
						Data: NodeData{
							ID:   "master_0",
							Role: "master",
							IP:   "192.168.3.10",
						},
					},
					{
						Data: NodeData{
							ID:            "slave_0",
							Role:          "slave",
							IP:            "192.168.3.20",
							Port:          "2404",
							CommonAddress: 1,
							SinglePoints: &IEC104PointConfig{
								Type:     "sequential",
								StartIOA: 1001,
								Values:   []int{1, 0, 1},
								TypeID:   "M_SP_NA_1",
							},
						},
					},
				},
				Edges: []Edge{
					{Data: EdgeData{ID: "edge_0", Source: "master_0", Target: "slave_0"}},
				},
			},
			level:    ERROR,
			wantLogs: 0,
		},
		{
			name: "Duplicate IDs",
			scenario: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{Data: NodeData{ID: "master_0", Role: "master", IP: "192.168.1.10"}},
					{Data: NodeData{ID: "master_0", Role: "slave", IP: "192.168.1.20"}},
				},
			},
			level:    ERROR,
			wantLogs: 1,
		},
		{
			name: "Same role edge connection",
			scenario: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{Data: NodeData{ID: "master_0", Role: "master", IP: "192.168.1.10"}},
					{Data: NodeData{ID: "master_1", Role: "master", IP: "192.168.1.11"}},
				},
				Edges: []Edge{
					{Data: EdgeData{ID: "edge_0", Source: "master_0", Target: "master_1"}},
				},
			},
			level:    ERROR,
			wantLogs: 1,
		},
		{
			name: "IP out of range",
			scenario: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{Data: NodeData{ID: "master_0", Role: "master", IP: "192.168.2.10"}},
				},
			},
			level:    ERROR,
			wantLogs: 1,
		},
		{
			name: "DNP3 slave without outstation_id",
			scenario: CytoscapeData{
				Protocol:  "dnp3",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{
						Data: NodeData{
							ID:   "slave_0",
							Role: "slave",
							IP:   "192.168.1.20",
						},
					},
				},
			},
			level:    WARNING,
			wantLogs: 4, // Missing outstation_id, master_id, no inputs, and node without link
		},
		{
			name: "IEC104 slave without common_address",
			scenario: CytoscapeData{
				Protocol:  "iec104",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{
						Data: NodeData{
							ID:   "slave_0",
							Role: "slave",
							IP:   "192.168.1.20",
						},
					},
				},
			},
			level:    WARNING,
			wantLogs: 3, // Missing common_address, no points, and node without link
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := ValidateCytoscapeScenario(tt.scenario, tt.level)
			if len(logs) != tt.wantLogs {
				t.Errorf("ValidateCytoscapeScenario() got %d logs, want %d. Logs: %v", len(logs), tt.wantLogs, logs)
			}
		})
	}
}

func TestParseCytoscapeJSON_Modbus(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []Node{
			{
				Data: NodeData{
					ID:   "master_0",
					Role: "master",
					Name: "Master Node",
					IP:   "192.168.1.10",
				},
			},
			{
				Data: NodeData{
					ID:               "slave_0",
					Role:             "slave",
					Name:             "Slave Node",
					IP:               "192.168.1.20",
					Port:             "502",
					SlaveID:          "1",
					HoldingRegisters: RegisterConfig{Type: "sequential", Values: "1,2,3"},
				},
			},
		},
		Edges: []Edge{
			{
				Data: EdgeData{
					ID:     "edge_0",
					Source: "master_0",
					Target: "slave_0",
					Messages: []Message{
						{
							Timestamp:    0,
							Recurrent:    true,
							Interval:     intPtr(5),
							FunctionCode: 3,
							StartAddress: 0,
							Count:        10,
						},
					},
				},
			},
		},
	}

	yaml, err := ParseCytoscapeJSON(data)
	if err != nil {
		t.Fatalf("ParseCytoscapeJSON() error = %v", err)
	}

	if yaml == "" {
		t.Error("ParseCytoscapeJSON() returned empty YAML")
	}

	// Verify protocol is in output
	if !contains(yaml, "protocol: modbus") {
		t.Error("ParseCytoscapeJSON() YAML missing protocol field")
	}

	// Verify nodes are in output
	if !contains(yaml, "master_0") || !contains(yaml, "slave_0") {
		t.Error("ParseCytoscapeJSON() YAML missing nodes")
	}
}

func TestParseCytoscapeJSON_DNP3(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "dnp3",
		IPNetwork: "192.168.2.0/24",
		Nodes: []Node{
			{
				Data: NodeData{
					ID:   "master_0",
					Role: "master",
					Name: "DNP3 Master",
					IP:   "192.168.2.10",
				},
			},
			{
				Data: NodeData{
					ID:           "slave_0",
					Role:         "slave",
					Name:         "DNP3 Outstation",
					IP:           "192.168.2.20",
					Port:         "20000",
					OutstationID: 1,
					MasterID:     2,
					AnalogInputs: &DNP3PointConfig{
						Count: 20,
						InitialValues: []map[string]interface{}{
							{"index": 0, "value": 100.0},
						},
					},
				},
			},
		},
		Edges: []Edge{
			{
				Data: EdgeData{
					ID:     "edge_0",
					Source: "master_0",
					Target: "slave_0",
					Messages: []Message{
						{
							Timestamp:     0,
							Recurrent:     true,
							Interval:      intPtr(5),
							OperationType: "read",
							Group:         30,
							Variation:     5,
							Index:         0,
							MasterID:      2,
							OutstationID:  1,
						},
					},
				},
			},
		},
	}

	yaml, err := ParseCytoscapeJSON(data)
	if err != nil {
		t.Fatalf("ParseCytoscapeJSON() error = %v", err)
	}

	if !contains(yaml, "protocol: dnp3") {
		t.Error("ParseCytoscapeJSON() YAML missing DNP3 protocol")
	}

	if !contains(yaml, "outstation_id") {
		t.Error("ParseCytoscapeJSON() YAML missing DNP3 specific fields")
	}
}

func TestParseCytoscapeJSON_IEC104(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "iec104",
		IPNetwork: "192.168.3.0/24",
		Nodes: []Node{
			{
				Data: NodeData{
					ID:   "master_0",
					Role: "master",
					Name: "IEC104 Client",
					IP:   "192.168.3.10",
				},
			},
			{
				Data: NodeData{
					ID:            "slave_0",
					Role:          "slave",
					Name:          "IEC104 Server",
					IP:            "192.168.3.20",
					Port:          "2404",
					CommonAddress: 1,
					T1:            15,
					T2:            10,
					T3:            20,
					SinglePoints: &IEC104PointConfig{
						Type:     "sequential",
						StartIOA: 1001,
						Values:   []int{1, 0, 1, 0},
						TypeID:   "M_SP_NA_1",
					},
				},
			},
		},
		Edges: []Edge{
			{
				Data: EdgeData{
					ID:     "edge_0",
					Source: "master_0",
					Target: "slave_0",
					Messages: []Message{
						{
							Timestamp:     0,
							Recurrent:     true,
							Interval:      intPtr(5),
							TypeID:        45,
							CommonAddress: 1,
							IOA:           1001,
							COT:           6,
							Value:         "1",
						},
					},
				},
			},
		},
	}

	yaml, err := ParseCytoscapeJSON(data)
	if err != nil {
		t.Fatalf("ParseCytoscapeJSON() error = %v", err)
	}

	if !contains(yaml, "protocol: iec104") {
		t.Error("ParseCytoscapeJSON() YAML missing IEC104 protocol")
	}

	if !contains(yaml, "common_address") {
		t.Error("ParseCytoscapeJSON() YAML missing IEC104 specific fields")
	}
}

func TestGenerateNetworkNodes_AllProtocols(t *testing.T) {
	tests := []struct {
		name        string
		protocol    Protocol
		masterNodes int
		slaveNodes  int
		wantMasters int
		wantSlaves  int
	}{
		{
			name:        "Modbus network",
			protocol:    MODBUS,
			masterNodes: 2,
			slaveNodes:  3,
			wantMasters: 2,
			wantSlaves:  3,
		},
		{
			name:        "DNP3 network",
			protocol:    DNP3,
			masterNodes: 1,
			slaveNodes:  2,
			wantMasters: 1,
			wantSlaves:  2,
		},
		{
			name:        "IEC104 network",
			protocol:    IEC104,
			masterNodes: 1,
			slaveNodes:  4,
			wantMasters: 1,
			wantSlaves:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseIP := net.ParseIP("192.168.1.10")
			nodes, edges := GenerateNetworkNodes(tt.protocol, baseIP, tt.masterNodes, tt.slaveNodes)

			masterCount := 0
			slaveCount := 0
			for _, node := range nodes {
				if node.Data.Role == "master" {
					masterCount++
				} else if node.Data.Role == "slave" {
					slaveCount++

					// Verify protocol-specific fields
					switch tt.protocol {
					case MODBUS:
						if node.Data.Port != "502" {
							t.Errorf("Modbus slave has wrong port: %v", node.Data.Port)
						}
						if node.Data.SlaveID != "1" {
							t.Errorf("Modbus slave has wrong slave_id: %v", node.Data.SlaveID)
						}
					case DNP3:
						if node.Data.Port != "20000" {
							t.Errorf("DNP3 slave has wrong port: %v", node.Data.Port)
						}
						if node.Data.OutstationID != "1" {
							t.Errorf("DNP3 slave has wrong outstation_id: %v", node.Data.OutstationID)
						}
						if node.Data.AnalogInputs == nil {
							t.Error("DNP3 slave missing analog_inputs")
						}
					case IEC104:
						if node.Data.Port != "2404" {
							t.Errorf("IEC104 slave has wrong port: %v", node.Data.Port)
						}
						if node.Data.CommonAddress != "1" {
							t.Errorf("IEC104 slave has wrong common_address: %v", node.Data.CommonAddress)
						}
						if node.Data.SinglePoints == nil {
							t.Error("IEC104 slave missing single_points")
						}
					}
				}
			}

			if masterCount != tt.wantMasters {
				t.Errorf("Got %d masters, want %d", masterCount, tt.wantMasters)
			}
			if slaveCount != tt.wantSlaves {
				t.Errorf("Got %d slaves, want %d", slaveCount, tt.wantSlaves)
			}

			if len(edges) != 0 {
				t.Errorf("Expected 0 edges for new network, got %d", len(edges))
			}
		})
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
		t.Errorf("CleanDictValues() clean field = %v, want 'hello world'", result["clean"])
	}

	if result["special"] != "testvalue" {
		t.Errorf("CleanDictValues() special field = %v, want 'testvalue'", result["special"])
	}

	if result["number"] != 123 {
		t.Errorf("CleanDictValues() number field = %v, want 123", result["number"])
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

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

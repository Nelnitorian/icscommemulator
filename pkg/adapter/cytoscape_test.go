package adapter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCytoscapeScenario(t *testing.T) {
	tests := []struct {
		name     string
		data     CytoscapeData
		level    Level
		expected []string
	}{
		{
			name: "valid scenario",
			data: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{Data: NodeData{ID: "master1", Role: "master", IP: "192.168.1.10"}},
					{Data: NodeData{ID: "slave1", Role: "slave", IP: "192.168.1.11"}},
				},
				Edges: []Edge{
					{Data: EdgeData{ID: "edge1", Source: "master1", Target: "slave1"}},
				},
			},
			level:    ERROR,
			expected: nil,
		},
		{
			name: "duplicate node IDs",
			data: CytoscapeData{
				Protocol:  "modbus",
				IPNetwork: "192.168.1.0/24",
				Nodes: []Node{
					{Data: NodeData{ID: "duplicate", Role: "master", IP: "192.168.1.10"}},
					{Data: NodeData{ID: "duplicate", Role: "slave", IP: "192.168.1.11"}},
				},
			},
			level:    ERROR,
			expected: []string{"[ERROR] All node IDs are not unique. Duplicates: [duplicate]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Nota: Ajustar esta llamada según la signatura real de ValidateCytoscapeScenario
			result := ValidateCytoscapeScenario(tt.data, tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateNetworkNodes(t *testing.T) {
	baseIP := net.ParseIP("192.168.1.10")
	nodes, edges := GenerateNetworkNodes(MODBUS, baseIP, 2, 3)

	// Test nodes count
	assert.Len(t, nodes, 5) // 2 masters + 3 slaves
	assert.Len(t, edges, 0) // No edges generated initially

	// Test masters
	masterCount := 0
	slaveCount := 0
	for _, node := range nodes {
		if node.Data.Role == "master" {
			masterCount++
			assert.Contains(t, node.Data.ID, "master_")
		} else if node.Data.Role == "slave" {
			slaveCount++
			assert.Contains(t, node.Data.ID, "slave_")
		}
	}

	assert.Equal(t, 2, masterCount)
	assert.Equal(t, 3, slaveCount)
}

func TestParseCytoscapeJSON(t *testing.T) {
	data := CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []Node{
			{Data: NodeData{
				ID:   "master1",
				Role: "master",
				IP:   "192.168.1.10",
				Messages: []Message{
					{
						Timestamp:    1,    // int en lugar de string
						Recurrent:    true,
						Interval:     func() *int { i := 5; return &i }(), // pointer a int
						FunctionCode: 3,    // int en lugar de string
						StartAddress: 0,
						Count:        10,
					},
				},
			}},
			{Data: NodeData{
				ID:      "slave1",
				Role:    "slave",
				IP:      "192.168.1.11",
				Port:    "502",
				SlaveID: "1",
				HoldingRegisters: RegisterConfig{  // ← Crear el struct correctamente
					Type:   "sequential",
					Values: "1,2,3,4,5",
				},
			}},

		},
		Edges: []Edge{
			{Data: EdgeData{
				ID:     "edge1",
				Source: "master1",
				Target: "slave1",
				Messages: []Message{
					{
						Timestamp:    1,
						Recurrent:    true,
						Interval:     func() *int { i := 5; return &i }(),
						FunctionCode: 3,
						StartAddress: 0,
						Count:        10,
					},
				},
			}},
		},
	}

	result, err := ParseCytoscapeJSON(data)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "protocol: modbus")
	assert.Contains(t, result, "ip_network: 192.168.1.0/24")
}

func TestCleanDictValues(t *testing.T) {
	input := map[string]interface{}{
		"valid":   "hello world",
		"numbers": "123",
		"special": "test@#$%invalid",
		"mixed":   "valid123_-chars",
	}

	result := CleanDictValues(input)
	
	assert.Equal(t, "hello world", result["valid"])
	assert.Equal(t, "123", result["numbers"])
	assert.Equal(t, "testinvalid", result["special"]) // Special chars removed
	assert.Equal(t, "valid123_-chars", result["mixed"])
}

// CORRECCIÓN COMPLETA: pkg/adapter/converter.go

package adapter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Converter struct{}

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) CytoscapeToSimplified(data CytoscapeData) (map[string]interface{}, error) {
	simpleNodes := []map[string]interface{}{}
	nodesByID := make(map[string]*NodeData)
	
	for _, node := range data.Nodes {
		nodeBytes, err := json.Marshal(node.Data)
		if err != nil {
			continue
		}
		
		var nodeMap map[string]interface{}
		if err := json.Unmarshal(nodeBytes, &nodeMap); err != nil {
			continue
		}
		
		simpleNodes = append(simpleNodes, nodeMap)
		nodeCopy := node.Data
		nodesByID[nodeCopy.ID] = &nodeCopy
	}

	masterMessages := make(map[string][]map[string]interface{})
	
	for _, edge := range data.Edges {
		sourceID := edge.Data.Source
		targetID := edge.Data.Target
		
		targetNode, exists := nodesByID[targetID]
		if !exists {
			continue
		}
		
		for _, msg := range edge.Data.Messages {
			msgMap := c.messageToMap(msg, targetNode)
			masterMessages[sourceID] = append(masterMessages[sourceID], msgMap)
		}
	}
	
	for i, nodeMap := range simpleNodes {
		if nodeID, ok := nodeMap["id"].(string); ok {
			if messages, exists := masterMessages[nodeID]; exists && len(messages) > 0 {
				simpleNodes[i]["messages"] = messages
			}
		}
	}
	
	return map[string]interface{}{
		"protocol":   data.Protocol,
		"ip_network": data.IPNetwork,
		"nodes":      simpleNodes,
	}, nil
}

func (c *Converter) messageToMap(msg Message, target *NodeData) map[string]interface{} {
	result := map[string]interface{}{
		"timestamp": msg.Timestamp,
		"recurrent": msg.Recurrent,
	}
	
	// CORRECCIÓN 1: Interval es *int
	if msg.Interval != nil && *msg.Interval > 0 {
		result["interval"] = *msg.Interval
	}
	
	// IP
	if msg.IP == "" {
		result["ip"] = target.IP
	} else {
		result["ip"] = msg.IP
	}
	
	// Puerto
	var targetPort int
	switch p := target.Port.(type) {
	case int:
		targetPort = p
	case float64:
		targetPort = int(p)
	case string:
		fmt.Sscanf(p, "%d", &targetPort)
	}
	
	if msg.Port == 0 && targetPort != 0 {
		result["port"] = targetPort
	} else if msg.Port != 0 {
		result["port"] = msg.Port
	}
	
	// MODBUS
	if msg.SlaveID > 0 {
		result["slave_id"] = msg.SlaveID
	}
	if msg.FunctionCode > 0 {
		result["function_code"] = msg.FunctionCode
	}
	
	// CORRECCIÓN 2: StartAddress es interface{} - convertir a int
	if msg.StartAddress != nil {
		startAddr, err := c.parseAddressValue(msg.StartAddress)
		if err == nil {
			result["start_address"] = startAddr
		}
	}
	
	if msg.Count > 0 {
		result["count"] = msg.Count
	}
	if len(msg.Values) > 0 {
		result["values"] = msg.Values
	}
	
	// DNP3
	if msg.OperationType != "" {
		result["operation_type"] = msg.OperationType
	}
	if msg.Group > 0 {
		result["group"] = msg.Group
	}
	if msg.Variation > 0 {
		result["variation"] = msg.Variation
	}
	if msg.Index > 0 {
		result["index"] = msg.Index
	}
	if msg.MasterID > 0 {
		result["master_id"] = msg.MasterID
	}
	if msg.OutstationID > 0 {
		result["outstation_id"] = msg.OutstationID
	}
	if msg.Value != "" {
		result["value"] = msg.Value
	}
	
	// IEC104
	if msg.TypeID > 0 {
		result["type_id"] = msg.TypeID
	}
	if msg.CommonAddress > 0 {
		result["common_address"] = msg.CommonAddress
	}
	if msg.IOA > 0 {
		result["ioa"] = msg.IOA
	}
	if msg.COT > 0 {
		result["cot"] = msg.COT
	}
	
	return result
}

// NUEVA FUNCIÓN: parseAddressValue maneja conversión de direcciones hex/dec
func (c *Converter) parseAddressValue(addr interface{}) (int, error) {
	switch v := addr.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		v = strings.TrimSpace(v)
		// Manejar formato hexadecimal
		if strings.HasPrefix(strings.ToLower(v), "0x") {
			val, err := strconv.ParseInt(v[2:], 16, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid hex address '%s': %w", v, err)
			}
			return int(val), nil
		}
		// Manejar formato decimal
		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal address '%s': %w", v, err)
		}
		return val, nil
	default:
		return 0, fmt.Errorf("invalid address type: %T", addr)
	}
}

func (c *Converter) SimplifiedToCytoscape(simplified map[string]interface{}) (CytoscapeData, error) {
	var result CytoscapeData
	
	if protocol, ok := simplified["protocol"].(string); ok {
		result.Protocol = protocol
	}
	if ipNetwork, ok := simplified["ip_network"].(string); ok {
		result.IPNetwork = ipNetwork
	}
	
	if nodesInterface, ok := simplified["nodes"]; ok {
		jsonData, _ := json.Marshal(nodesInterface)
		var nodes []NodeData
		json.Unmarshal(jsonData, &nodes)
		
		for _, nodeData := range nodes {
			result.Nodes = append(result.Nodes, Node{Data: nodeData})
		}
	}
	
	result.Edges = []Edge{}
	return result, nil
}

package adapter

import (
	"encoding/json"
	"fmt"
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

	if msg.Interval != nil && *msg.Interval > 0 {
		result["interval"] = *msg.Interval
	}

	if msg.IP == "" {
		result["ip"] = target.IP
	} else {
		result["ip"] = msg.IP
	}

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

	if msg.SlaveID > 0 {
		result["slave_id"] = msg.SlaveID
	}
	if msg.FunctionCode > 0 {
		result["function_code"] = msg.FunctionCode
	}

	if msg.StartAddress != nil {
		startAddr, err := parseAddressValue(msg.StartAddress)
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

	if msg.OperationType != "" {
		result["operation_type"] = msg.OperationType
	}
	if msg.Group > 0 {
		result["group"] = msg.Group
	}
	if msg.Variation > 0 {
		result["variation"] = msg.Variation
	}
	if msg.Index > 0 || msg.OperationType == "poll_group_variation_index" ||
		msg.OperationType == "send_binary_command" ||
		msg.OperationType == "send_analog_command_float32" ||
		msg.OperationType == "send_analog_command_int16" ||
		msg.OperationType == "send_analog_command_int32" ||
		msg.OperationType == "send_analog_command_double64" {
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

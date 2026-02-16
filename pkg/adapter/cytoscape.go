package adapter

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
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
	DNP3   Protocol = "dnp3"
	IEC104 Protocol = "iec104"
)

type DNP3Point struct {
	Value interface{} `json:"value" yaml:"value"`
	Flags int         `json:"flags,omitempty" yaml:"flags,omitempty"`
}

type IEC104Point struct {
	IOA      int         `json:"ioa" yaml:"ioa"`
	Value    interface{} `json:"value" yaml:"value"`
	ReportMS int         `json:"report_ms" yaml:"report_ms"`
}

type NodeData struct {
	ID    string      `json:"id" yaml:"id"`
	Label string      `json:"label,omitempty" yaml:"label,omitempty"`
	Role  string      `json:"role" yaml:"role"`
	Name  string      `json:"name" yaml:"name"`
	IP    string      `json:"ip" yaml:"ip"`
	Port  interface{} `json:"port,omitempty" yaml:"port,omitempty"`

	TickRateMS        int   `json:"tick_rate_ms,omitempty" yaml:"tick_rate_ms,omitempty"`
	SelectTimeoutMS   int   `json:"select_timeout_ms,omitempty" yaml:"select_timeout_ms,omitempty"`
	MaxConnections    int   `json:"max_connections,omitempty" yaml:"max_connections,omitempty"`
	AuthorizedMasters []int `json:"authorized_masters,omitempty" yaml:"authorized_masters,omitempty"`

	SlaveID  interface{}    `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	Comment  string         `json:"comment,omitempty" yaml:"comment,omitempty"`
	Messages []Message      `json:"messages,omitempty" yaml:"messages,omitempty"`
	Attacks  []AttackConfig `json:"attacks,omitempty" yaml:"attacks,omitempty"`

	HoldingRegisters map[string]int `json:"holding_registers,omitempty" yaml:"holding_registers,omitempty"`
	Coils            map[string]int `json:"coils,omitempty" yaml:"coils,omitempty"`
	DiscreteInputs   map[string]int `json:"discrete_inputs,omitempty" yaml:"discrete_inputs,omitempty"`
	InputRegisters   map[string]int `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`

	OutstationID       interface{}          `json:"outstation_id,omitempty" yaml:"outstation_id,omitempty"`
	MasterID           interface{}          `json:"master_id,omitempty" yaml:"master_id,omitempty"`
	AnalogInputs       map[string]DNP3Point `json:"analog_inputs,omitempty" yaml:"analog_inputs,omitempty"`
	BinaryInputs       map[string]DNP3Point `json:"binary_inputs,omitempty" yaml:"binary_inputs,omitempty"`
	AnalogOutputStatus map[string]DNP3Point `json:"analog_output_status,omitempty" yaml:"analog_output_status,omitempty"`
	BinaryOutputStatus map[string]DNP3Point `json:"binary_output_status,omitempty" yaml:"binary_output_status,omitempty"`
	Simulation         *SimulationConfig    `json:"simulation,omitempty" yaml:"simulation,omitempty"`

	CommonAddress      interface{}            `json:"common_address,omitempty" yaml:"common_address,omitempty"`
	T1                 interface{}            `json:"t1,omitempty" yaml:"t1,omitempty"`
	T2                 interface{}            `json:"t2,omitempty" yaml:"t2,omitempty"`
	T3                 interface{}            `json:"t3,omitempty" yaml:"t3,omitempty"`
	K                  interface{}            `json:"k,omitempty" yaml:"k,omitempty"`
	W                  interface{}            `json:"w,omitempty" yaml:"w,omitempty"`
	SinglePoints       map[string]IEC104Point `json:"single_points,omitempty" yaml:"single_points,omitempty"`
	DoublePoints       map[string]IEC104Point `json:"double_points,omitempty" yaml:"double_points,omitempty"`
	MeasuredScaled     map[string]IEC104Point `json:"measured_scaled,omitempty" yaml:"measured_scaled,omitempty"`
	MeasuredNormalized map[string]IEC104Point `json:"measured_normalized,omitempty" yaml:"measured_normalized,omitempty"`
	MeasuredShort      map[string]IEC104Point `json:"measured_short,omitempty" yaml:"measured_short,omitempty"`

	Identity map[string]interface{} `json:"identity,omitempty" yaml:"identity,omitempty"`

	Position   *Position `json:"position,omitempty" yaml:"-"`
	Group      string    `json:"group,omitempty" yaml:"-"`
	Removed    bool      `json:"removed,omitempty" yaml:"-"`
	Selected   bool      `json:"selected,omitempty" yaml:"-"`
	Selectable bool      `json:"selectable,omitempty" yaml:"-"`
	Locked     bool      `json:"locked,omitempty" yaml:"-"`
	Grabbable  bool      `json:"grabbable,omitempty" yaml:"-"`
	Pannable   bool      `json:"pannable,omitempty" yaml:"-"`
	Classes    string    `json:"classes,omitempty" yaml:"-"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type SimulationConfig struct {
	Enabled  bool `json:"enabled" yaml:"enabled"`
	Interval int  `json:"interval,omitempty" yaml:"interval,omitempty"`
}

type Node struct {
	Data     NodeData    `json:"data" yaml:"data"`
	Classes  NodeClasses `json:"classes,omitempty" yaml:"classes,omitempty"`
	Position *Position   `json:"position,omitempty" yaml:"-"`
}

type EdgeData struct {
	ID             string    `json:"id" yaml:"id"`
	Source         string    `json:"source" yaml:"source"`
	Target         string    `json:"target" yaml:"target"`
	Messages       []Message `json:"messages,omitempty" yaml:"messages,omitempty"`
	Timestamp      string    `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	FunctionCode   string    `json:"function_code,omitempty" yaml:"function_code,omitempty"`
	StartAddress   int       `json:"start_address,omitempty" yaml:"start_address,omitempty"`
	InputRegisters string    `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`
}

type Edge struct {
	Data EdgeData `json:"data" yaml:"data"`
}

type Message struct {
	Timestamp    float64       `json:"timestamp" yaml:"timestamp"`
	Recurrent    bool          `json:"recurrent" yaml:"recurrent"`
	Interval     *float64      `json:"interval,omitempty" yaml:"interval,omitempty"`
	IP           string        `json:"ip" yaml:"ip"`
	Port         int           `json:"port" yaml:"port"`
	SlaveID      int           `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	FunctionCode int           `json:"function_code,omitempty" yaml:"function_code,omitempty"`
	StartAddress interface{}   `json:"start_address,omitempty" yaml:"start_address,omitempty"`
	Count        int           `json:"count,omitempty" yaml:"count,omitempty"`
	Values       []interface{} `json:"values,omitempty" yaml:"values,omitempty"`

	OperationType string `json:"operation_type,omitempty" yaml:"operation_type,omitempty"`
	Group         int    `json:"group,omitempty" yaml:"group,omitempty"`
	Variation     int    `json:"variation,omitempty" yaml:"variation,omitempty"`
	Index         int    `json:"index,omitempty" yaml:"index,omitempty"`
	MasterID      int    `json:"master_id,omitempty" yaml:"master_id,omitempty"`
	OutstationID  int    `json:"outstation_id,omitempty" yaml:"outstation_id,omitempty"`
	Value         string `json:"value,omitempty" yaml:"value,omitempty"`

	TypeID        int `json:"type_id,omitempty" yaml:"type_id,omitempty"`
	CommonAddress int `json:"common_address,omitempty" yaml:"common_address,omitempty"`
	IOA           int `json:"ioa,omitempty" yaml:"ioa,omitempty"`
	COT           int `json:"cot,omitempty" yaml:"cot,omitempty"`
}

type CytoscapeData struct {
	Protocol  string `json:"protocol" yaml:"protocol"`
	IPNetwork string `json:"ip_network" yaml:"ip_network"`
	Nodes     []Node `json:"nodes" yaml:"nodes"`
	Edges     []Edge `json:"edges" yaml:"edges"`
}

type NodeClasses string

func (c *NodeClasses) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*c = NodeClasses(asString)
		return nil
	}

	var asArray []string
	if err := json.Unmarshal(data, &asArray); err == nil {
		*c = NodeClasses(strings.Join(asArray, " "))
		return nil
	}

	var asAny []interface{}
	if err := json.Unmarshal(data, &asAny); err == nil {
		values := make([]string, 0, len(asAny))
		for _, item := range asAny {
			if str, ok := item.(string); ok {
				values = append(values, str)
			}
		}
		*c = NodeClasses(strings.Join(values, " "))
		return nil
	}

	return fmt.Errorf("invalid classes value: %s", string(data))
}

type YAMLData struct {
	Protocol  string     `yaml:"protocol"`
	IPNetwork string     `yaml:"ip_network"`
	Nodes     []NodeData `yaml:"nodes"`
}

func ValidateCytoscapeScenario(data CytoscapeData, level Level) []string {
	var logs []string

	if level >= ERROR {
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
		switch Protocol(data.Protocol) {
		case MODBUS:
			logs = append(logs, validateModbusNodes(data.Nodes)...)
		case DNP3:
			logs = append(logs, validateDNP3Nodes(data.Nodes)...)
		case IEC104:
			logs = append(logs, validateIEC104Nodes(data.Nodes)...)
		}

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

func validateModbusNodes(nodes []Node) []string {
	var logs []string
	for _, node := range nodes {
		if node.Data.Role == "slave" {
			hasRegisters := len(node.Data.HoldingRegisters) > 0 ||
				len(node.Data.Coils) > 0 ||
				len(node.Data.DiscreteInputs) > 0 ||
				len(node.Data.InputRegisters) > 0
			if !hasRegisters {
				logs = append(logs, fmt.Sprintf("[WARNING] Modbus slave node %s has no defined registers", node.Data.ID))
			}
		}
	}
	return logs
}

func validateDNP3Nodes(nodes []Node) []string {
	var logs []string
	for _, node := range nodes {
		if node.Data.Role == "slave" {
			if node.Data.OutstationID == nil {
				logs = append(logs, fmt.Sprintf("[WARNING] DNP3 slave node %s missing outstation_id", node.Data.ID))
			}
			if node.Data.MasterID == nil {
				logs = append(logs, fmt.Sprintf("[WARNING] DNP3 slave node %s missing master_id", node.Data.ID))
			}

			if len(node.Data.AnalogInputs) == 0 && len(node.Data.BinaryInputs) == 0 {
				logs = append(logs, fmt.Sprintf("[WARNING] DNP3 slave node %s has no inputs configured", node.Data.ID))
			}
		}
	}
	return logs
}

func validateIEC104Nodes(nodes []Node) []string {
	var logs []string
	for _, node := range nodes {
		if node.Data.Role == "slave" {
			if node.Data.CommonAddress == nil {
				logs = append(logs, fmt.Sprintf("[WARNING] IEC104 slave node %s missing common_address", node.Data.ID))
			}

			hasPoints := len(node.Data.SinglePoints) > 0 ||
				len(node.Data.DoublePoints) > 0 ||
				len(node.Data.MeasuredScaled) > 0 ||
				len(node.Data.MeasuredNormalized) > 0 ||
				len(node.Data.MeasuredShort) > 0

			if !hasPoints {
				logs = append(logs, fmt.Sprintf("[WARNING] IEC104 slave node %s has no points configured", node.Data.ID))
			}
		}
	}
	return logs
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

	messagesDict := make(map[string][]Message)
	for _, node := range data.Nodes {
		if node.Data.Role == "master" {
			messagesDict[node.Data.ID] = []Message{}
		}
	}

	for _, edge := range data.Edges {
		sourceNode := findFirstMatchingNode(data.Nodes, edge.Data.Source)
		targetNode := findFirstMatchingNode(data.Nodes, edge.Data.Target)
		if sourceNode == nil || targetNode == nil {
			continue
		}

		masterNode := sourceNode
		slaveNode := targetNode
		if sourceNode.Data.Role == "master" && targetNode.Data.Role == "slave" {
			masterNode = sourceNode
			slaveNode = targetNode
		} else if sourceNode.Data.Role == "slave" && targetNode.Data.Role == "master" {
			masterNode = targetNode
			slaveNode = sourceNode
		}

		for _, message := range edge.Data.Messages {
			port := 502
			if Protocol(data.Protocol) == DNP3 {
				port = 20000
			} else if Protocol(data.Protocol) == IEC104 {
				port = 2404
			}

			if p, ok := slaveNode.Data.Port.(string); ok {
				if pInt, err := parseIntFromString(p); err == nil {
					port = pInt
				}
			} else if pInt, ok := slaveNode.Data.Port.(int); ok {
				port = pInt
			}

			msg := Message{
				Timestamp: message.Timestamp,
				Recurrent: message.Recurrent,
				Interval:  message.Interval,
				IP:        slaveNode.Data.IP,
				Port:      port,
			}

			switch Protocol(data.Protocol) {
			case MODBUS:
				slaveID := 1
				if s, ok := slaveNode.Data.SlaveID.(string); ok {
					if sInt, err := parseIntFromString(s); err == nil {
						slaveID = sInt
					}
				} else if sInt, ok := slaveNode.Data.SlaveID.(int); ok {
					slaveID = sInt
				}
				msg.SlaveID = slaveID
				msg.FunctionCode = message.FunctionCode
				msg.Count = message.Count
				msg.Values = message.Values
				if message.StartAddress != nil {
					addr, err := parseAddressValue(message.StartAddress)
					if err != nil {
						return "", fmt.Errorf("error parsing start_address in message: %w", err)
					}
					msg.StartAddress = addr
				}

			case DNP3:
				msg.OperationType = message.OperationType
				msg.Group = message.Group
				msg.Variation = message.Variation
				msg.Index = message.Index
				msg.MasterID = message.MasterID
				msg.OutstationID = message.OutstationID
				msg.Value = message.Value
			case IEC104:
				msg.TypeID = message.TypeID
				msg.CommonAddress = message.CommonAddress
				msg.IOA = message.IOA
				msg.COT = message.COT
				msg.Value = message.Value
			}

			messagesDict[masterNode.Data.ID] = append(messagesDict[masterNode.Data.ID], msg)
		}
	}

	for _, node := range data.Nodes {
		nodeData := node.Data
		if nodeData.Role == "master" {
			nodeData.Messages = messagesDict[nodeData.ID]
		}

		if nodeData.Role == "slave" {
			if port, ok := nodeData.Port.(string); ok {
				if p, err := parseIntFromString(port); err == nil {
					nodeData.Port = p
				}
			}

			switch Protocol(data.Protocol) {
			case MODBUS:
				if slaveID, ok := nodeData.SlaveID.(string); ok {
					if s, err := parseIntFromString(slaveID); err == nil {
						nodeData.SlaveID = s
					}
				}
			case DNP3:
				if outstationID, ok := nodeData.OutstationID.(string); ok {
					if o, err := parseIntFromString(outstationID); err == nil {
						nodeData.OutstationID = o
					}
				}
				if masterID, ok := nodeData.MasterID.(string); ok {
					if m, err := parseIntFromString(masterID); err == nil {
						nodeData.MasterID = m
					}
				}
			case IEC104:
				if commonAddr, ok := nodeData.CommonAddress.(string); ok {
					if c, err := parseIntFromString(commonAddr); err == nil {
						nodeData.CommonAddress = c
					}
				}
			}
		}

		if nodeData.Identity != nil {
			nodeData.Identity = CleanDictValues(nodeData.Identity)
		}

		yamlData.Nodes = append(yamlData.Nodes, nodeData)
	}

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
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// GenerateNetworkNodes generates master and slave nodes for a given protocol
func GenerateNetworkNodes(proto Protocol, ipBase net.IP, masterNodes, slaveNodes int) ([]Node, []Edge) {
	nodes := []Node{}
	edges := []Edge{}

	currentIP := ipBase

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

	for i := 0; i < slaveNodes; i++ {
		data := NodeData{
			ID:    fmt.Sprintf("slave_%d", i),
			Label: "data(name)",
			Role:  "slave",
			Name:  fmt.Sprintf("slave_%d", i),
			IP:    currentIP.String(),
		}

		switch proto {
		case MODBUS:
			data.Port = "502"
			data.SlaveID = "1"
			data.Comment = ""
			data.HoldingRegisters = make(map[string]int)
			data.Coils = make(map[string]int)
			data.DiscreteInputs = make(map[string]int)
			data.InputRegisters = make(map[string]int)

			data.Identity = map[string]interface{}{
				"major_minor_revision":  "",
				"model_name":            "",
				"product_code":          "",
				"product_name":          "",
				"user_application_name": "",
				"vendor_name":           "",
				"vendor_url":            "",
			}
		case DNP3:
			data.Port = "20000"
			data.OutstationID = "1"
			data.MasterID = "2"
			data.AnalogInputs = make(map[string]DNP3Point)
			data.BinaryInputs = make(map[string]DNP3Point)
			data.AnalogOutputStatus = make(map[string]DNP3Point)
			data.BinaryOutputStatus = make(map[string]DNP3Point)

			data.Simulation = &SimulationConfig{
				Enabled: false,
			}
		case IEC104:
			data.Port = "2404"
			data.CommonAddress = "1"
			data.T1 = 15
			data.T2 = 10
			data.T3 = 20
			data.K = 12
			data.W = 8
			data.TickRateMS = 100
			data.SelectTimeoutMS = 10000
			data.MaxConnections = 5

			data.SinglePoints = make(map[string]IEC104Point)
			data.DoublePoints = make(map[string]IEC104Point)
			data.MeasuredScaled = make(map[string]IEC104Point)
			data.MeasuredNormalized = make(map[string]IEC104Point)
			data.MeasuredShort = make(map[string]IEC104Point)

			data.Identity = map[string]interface{}{
				"station_name": "",
				"location":     "",
				"description":  "",
				"version":      "",
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
	result := make(net.IP, len(ip))
	copy(result, ip)
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] > 0 {
			break
		}
	}
	return result
}

func parseAddressValue(addr interface{}) (int, error) {
	switch v := addr.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		v = strings.TrimSpace(v)
		if strings.HasPrefix(strings.ToLower(v), "0x") {
			val, err := strconv.ParseInt(v[2:], 16, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid hex address '%s': %w", v, err)
			}
			return int(val), nil
		}
		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal address '%s': %w", v, err)
		}
		return val, nil
	default:
		return 0, fmt.Errorf("invalid address type: %T", addr)
	}
}

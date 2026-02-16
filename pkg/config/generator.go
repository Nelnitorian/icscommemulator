package config

import (
	"fmt"
	"icscommemulator/pkg/logger"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Message represents a message configuration for master nodes (generic)
type Message struct {
	Timestamp float64 `json:"timestamp" yaml:"timestamp"`
	Recurrent bool    `json:"recurrent" yaml:"recurrent"`
	Interval  float64 `json:"interval,omitempty" yaml:"interval,omitempty"`
	IP        string `json:"ip" yaml:"ip"`
	Port      int    `json:"port" yaml:"port"`

	// MODBUS specific
	SlaveID      int           `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	FunctionCode int           `json:"function_code,omitempty" yaml:"function_code,omitempty"`
	StartAddress int           `json:"start_address,omitempty" yaml:"start_address,omitempty"`
	Count        int           `json:"count,omitempty" yaml:"count,omitempty"`
	Values       []interface{} `json:"values,omitempty" yaml:"values,omitempty"`

	// DNP3 specific
	OperationType string `json:"operation_type,omitempty" yaml:"operation_type,omitempty"`
	Group         int    `json:"group,omitempty" yaml:"group,omitempty"`
	Variation     int    `json:"variation,omitempty" yaml:"variation,omitempty"`
	Index         int    `json:"index,omitempty" yaml:"index,omitempty"`
	MasterID      int    `json:"master_id,omitempty" yaml:"master_id,omitempty"`
	OutstationID  int    `json:"outstation_id,omitempty" yaml:"outstation_id,omitempty"`
	Value         string `json:"value,omitempty" yaml:"value,omitempty"`

	// IEC104 specific
	TypeID        int `json:"type_id,omitempty" yaml:"type_id,omitempty"`
	CommonAddress int `json:"common_address,omitempty" yaml:"common_address,omitempty"`
	IOA           int `json:"ioa,omitempty" yaml:"ioa,omitempty"`
	COT           int `json:"cot,omitempty" yaml:"cot,omitempty"`
}

// Node represents a node in the scenario
type Node struct {
	Role     string    `json:"role" yaml:"role"`
	Messages []Message `json:"messages,omitempty" yaml:"messages,omitempty"`

	// Additional fields that might be present
	ID      string      `json:"id,omitempty" yaml:"id,omitempty"`
	Name    string      `json:"name,omitempty" yaml:"name,omitempty"`
	Label   string      `json:"label,omitempty" yaml:"label,omitempty"`
	Comment string      `json:"comment,omitempty" yaml:"comment,omitempty"`
	IP      string      `json:"ip,omitempty" yaml:"ip,omitempty"`
	Port    interface{} `json:"port,omitempty" yaml:"port,omitempty"`

	// MODBUS specific
	SlaveID          interface{}            `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	HoldingRegisters map[string]interface{} `json:"holding_registers,omitempty" yaml:"holding_registers,omitempty"`
	Coils            map[string]interface{} `json:"coils,omitempty" yaml:"coils,omitempty"`
	DiscreteInputs   map[string]interface{} `json:"discrete_inputs,omitempty" yaml:"discrete_inputs,omitempty"`
	InputRegisters   map[string]interface{} `json:"input_registers,omitempty" yaml:"input_registers,omitempty"`

	// DNP3 specific
	OutstationID       interface{}            `json:"outstation_id,omitempty" yaml:"outstation_id,omitempty"`
	MasterID           interface{}            `json:"master_id,omitempty" yaml:"master_id,omitempty"`
	AnalogInputs       map[string]interface{} `json:"analog_inputs,omitempty" yaml:"analog_inputs,omitempty"`
	BinaryInputs       map[string]interface{} `json:"binary_inputs,omitempty" yaml:"binary_inputs,omitempty"`
	AnalogOutputStatus map[string]interface{} `json:"analog_output_status,omitempty" yaml:"analog_output_status,omitempty"`
	BinaryOutputStatus map[string]interface{} `json:"binary_output_status,omitempty" yaml:"binary_output_status,omitempty"`
	Simulation         map[string]interface{} `json:"simulation,omitempty" yaml:"simulation,omitempty"`

	// IEC104 specific
	CommonAddress      interface{}            `json:"common_address,omitempty" yaml:"common_address,omitempty"`
	T1                 interface{}            `json:"t1,omitempty" yaml:"t1,omitempty"`
	T2                 interface{}            `json:"t2,omitempty" yaml:"t2,omitempty"`
	T3                 interface{}            `json:"t3,omitempty" yaml:"t3,omitempty"`
	K                  interface{}            `json:"k,omitempty" yaml:"k,omitempty"`
	W                  interface{}            `json:"w,omitempty" yaml:"w,omitempty"`
	SinglePoints       map[string]interface{} `json:"single_points,omitempty" yaml:"single_points,omitempty"`
	DoublePoints       map[string]interface{} `json:"double_points,omitempty" yaml:"double_points,omitempty"`
	MeasuredScaled     map[string]interface{} `json:"measured_scaled,omitempty" yaml:"measured_scaled,omitempty"`
	MeasuredNormalized map[string]interface{} `json:"measured_normalized,omitempty" yaml:"measured_normalized,omitempty"`

	// IEC104 New Fields for hierarchical YAML
	TickRateMS        int   `json:"tick_rate_ms,omitempty" yaml:"tick_rate_ms,omitempty"`
	SelectTimeoutMS   int   `json:"select_timeout_ms,omitempty" yaml:"select_timeout_ms,omitempty"`
	MaxConnections    int   `json:"max_connections,omitempty" yaml:"max_connections,omitempty"`
	AuthorizedMasters []int `json:"authorized_masters,omitempty" yaml:"authorized_masters,omitempty"`

	// Comandos y Setpoints para IEC104 (mapeados genéricamente)
	SingleCommands     map[string]interface{} `json:"single_commands,omitempty" yaml:"single_commands,omitempty"`
	SetpointShort      map[string]interface{} `json:"setpoint_short,omitempty" yaml:"setpoint_short,omitempty"`
	MeasuredShort      map[string]interface{} `json:"measured_short,omitempty" yaml:"measured_short,omitempty"`
	MeasuredShortTime  map[string]interface{} `json:"measured_short_time,omitempty" yaml:"measured_short_time,omitempty"`
	SinglePointsTime   map[string]interface{} `json:"single_points_time,omitempty" yaml:"single_points_time,omitempty"`
	SingleCommandsTime map[string]interface{} `json:"single_commands_time,omitempty" yaml:"single_commands_time,omitempty"`
	SetpointShortTime  map[string]interface{} `json:"setpoint_short_time,omitempty" yaml:"setpoint_short_time,omitempty"`

	// Common
	Identity map[string]interface{} `json:"identity,omitempty" yaml:"identity,omitempty"`
}

// Scenario represents the complete scenario configuration
type Scenario struct {
	Protocol string `json:"protocol" yaml:"protocol"`
	Nodes    []Node `json:"nodes" yaml:"nodes"`
}

// MasterConfig is a unified master schedule config (YAML)
type MasterConfig struct {
	Protocol string    `yaml:"protocol"`
	Messages []Message `yaml:"messages,omitempty"`
}

// SlaveConfig is a unified slave config wrapper (YAML)
type SlaveConfig struct {
	Protocol string      `yaml:"protocol"`
	Node     interface{} `yaml:"node"`
}

// Generator manages scenario configuration file generation
type Generator struct {
	scenario   Scenario
	configPath string
	protocol   string
}

// NewGenerator creates a new scenario configuration generator
func NewGenerator(scenario Scenario, configPath string) *Generator {
	return &Generator{
		scenario:   scenario,
		configPath: configPath,
		protocol:   scenario.Protocol,
	}
}

// ConvertToInt attempts to convert a value to an integer
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

func normalizeRegisterConfig(raw map[string]interface{}) ModbusRegisterConfig {
	if raw == nil {
		return ModbusRegisterConfig{Type: "sparse", Values: map[string]interface{}{}}
	}

	if typ, ok := raw["type"].(string); ok {
		values := raw["values"]
		if values == nil {
			values = map[string]interface{}{}
		}
		return ModbusRegisterConfig{Type: typ, Values: values}
	}

	// Default: sparse map from address -> value
	return ModbusRegisterConfig{
		Type:   "sparse",
		Values: raw,
	}
}

func buildModbusSlaveConfig(slave Node) (ModbusSlaveConfig, error) {
	config := ModbusSlaveConfig{
		IP:               slave.IP,
		Port:             502,
		SlaveID:          1,
		DiscreteInputs:   normalizeRegisterConfig(slave.DiscreteInputs),
		Coils:            normalizeRegisterConfig(slave.Coils),
		InputRegisters:   normalizeRegisterConfig(slave.InputRegisters),
		HoldingRegisters: normalizeRegisterConfig(slave.HoldingRegisters),
		Identity:         slave.Identity,
	}

	if p, ok := ConvertToInt(slave.Port).(int); ok {
		config.Port = p
	}
	if sid, ok := ConvertToInt(slave.SlaveID).(int); ok {
		config.SlaveID = sid
	}

	return config, nil
}

// CraftMaster creates configuration files for master nodes
func (g *Generator) CraftMaster(messages []Message, index int) error {
	masterDir := filepath.Join(g.configPath, "masters", strconv.Itoa(index))
	err := os.MkdirAll(masterDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create master directory: %w", err)
	}

	yamlFile := filepath.Join(masterDir, "master.yaml")
	file, err := os.Create(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to create master YAML file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	defer encoder.Close()

	config := MasterConfig{
		Protocol: g.protocol,
		Messages: messages,
	}

	return encoder.Encode(config)
}

// Estructuras auxiliares para el formato YAML específico de IEC104
type IEC104ProtocolParams struct {
	T1 int `yaml:"t1"`
	T2 int `yaml:"t2"`
	T3 int `yaml:"t3"`
	K  int `yaml:"k"`
	W  int `yaml:"w"`
}

type IEC104Station struct {
	CommonAddress int                      `yaml:"common_address"`
	Points        map[string][]interface{} `yaml:"points"`
}

type IEC104SlaveConfig struct {
	IP                 string               `yaml:"ip"`
	Port               int                  `yaml:"port"`
	TickRateMS         int                  `yaml:"tick_rate_ms"`
	SelectTimeoutMS    int                  `yaml:"select_timeout_ms"`
	MaxConnections     int                  `yaml:"max_connections"`
	ProtocolParameters IEC104ProtocolParams `yaml:"protocol_parameters"`
	Stations           []IEC104Station      `yaml:"stations"`
	AuthorizedMasters  []int                `yaml:"authorized_masters"`
}

type ModbusRegisterConfig struct {
	Type   string      `yaml:"type"`
	Values interface{} `yaml:"values"`
}

type ModbusSlaveConfig struct {
	IP               string                 `yaml:"ip"`
	Port             int                    `yaml:"port"`
	SlaveID          int                    `yaml:"slave_id"`
	DiscreteInputs   ModbusRegisterConfig   `yaml:"discrete_inputs"`
	Coils            ModbusRegisterConfig   `yaml:"coils"`
	InputRegisters   ModbusRegisterConfig   `yaml:"input_registers"`
	HoldingRegisters ModbusRegisterConfig   `yaml:"holding_registers"`
	Identity         map[string]interface{} `yaml:"identity,omitempty"`
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

	yamlFile := filepath.Join(slaveDir, "slave.yaml")
	file, err := os.Create(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2) // Indentación bonita
	defer encoder.Close()

	// LOGICA ESPECIFICA PARA IEC104 (Formato Jerárquico)
	if g.protocol == "iec104" {
		config := IEC104SlaveConfig{
			IP:                slave.IP,
			TickRateMS:        slave.TickRateMS,
			SelectTimeoutMS:   slave.SelectTimeoutMS,
			MaxConnections:    slave.MaxConnections,
			AuthorizedMasters: slave.AuthorizedMasters,
		}

		// Convertir Puerto
		if p, ok := ConvertToInt(slave.Port).(int); ok {
			config.Port = p
		}

		// Protocol Parameters
		if t1, ok := ConvertToInt(slave.T1).(int); ok {
			config.ProtocolParameters.T1 = t1
		}
		if t2, ok := ConvertToInt(slave.T2).(int); ok {
			config.ProtocolParameters.T2 = t2
		}
		if t3, ok := ConvertToInt(slave.T3).(int); ok {
			config.ProtocolParameters.T3 = t3
		}
		if k, ok := ConvertToInt(slave.K).(int); ok {
			config.ProtocolParameters.K = k
		}
		if w, ok := ConvertToInt(slave.W).(int); ok {
			config.ProtocolParameters.W = w
		}

		// Stations & Points
		// Creamos una única estación usando el CommonAddress del nodo
		commonAddr := 1 // Default
		if ca, ok := ConvertToInt(slave.CommonAddress).(int); ok {
			commonAddr = ca
		}

		points := make(map[string][]interface{})

		// Helpers para convertir mapas planos a listas de objetos
		addPoints := func(key string, source map[string]interface{}) {
			if len(source) > 0 {
				list := convertMapToList(source)
				if len(list) > 0 {
					points[key] = list
				}
			}
		}

		addPoints("single_points", slave.SinglePoints)
		addPoints("measured_scaled", slave.MeasuredScaled)
		addPoints("measured_short", slave.MeasuredShort)
		addPoints("measured_normalized", slave.MeasuredNormalized)
		addPoints("single_commands", slave.SingleCommands)
		addPoints("setpoint_short", slave.SetpointShort)
		addPoints("measured_short_time", slave.MeasuredShortTime)
		addPoints("single_points_time", slave.SinglePointsTime)
		addPoints("single_commands_time", slave.SingleCommandsTime)
		addPoints("setpoint_short_time", slave.SetpointShortTime)

		config.Stations = []IEC104Station{
			{
				CommonAddress: commonAddr,
				Points:        points,
			},
		}

		return encoder.Encode(SlaveConfig{
			Protocol: g.protocol,
			Node:     config,
		})
	}

	// MODBUS / DNP3: Wrap node config to keep a consistent YAML format
	if g.protocol == "modbus" {
		modbusNode, err := buildModbusSlaveConfig(slave)
		if err != nil {
			return err
		}
		return encoder.Encode(SlaveConfig{
			Protocol: g.protocol,
			Node:     modbusNode,
		})
	}

	// DNP3 (left as flat node until protocol migration)
	slaveCopy := make(map[string]interface{})
	slaveData, err := yaml.Marshal(slave)
	if err != nil {
		return fmt.Errorf("failed to marshal slave data: %w", err)
	}

	if err := yaml.Unmarshal(slaveData, &slaveCopy); err != nil {
		return fmt.Errorf("failed to unmarshal slave data: %w", err)
	}

	fieldsToRemove := []string{"comment", "label", "role", "name", "id"}
	for _, key := range fieldsToRemove {
		delete(slaveCopy, key)
	}

	return encoder.Encode(SlaveConfig{
		Protocol: g.protocol,
		Node:     slaveCopy,
	})
}

// convertMapToList convierte un map[string]interface{} (donde interface es un map de propiedades)
// a una lista ordenada por IOA para que el YAML quede limpio.
func convertMapToList(source map[string]interface{}) []interface{} {
	var list []interface{}

	// Para ordenar la salida, necesitamos extraer las claves o IOAs
	type item struct {
		ioa int
		val map[string]interface{}
	}
	var items []item

	for k, v := range source {
		props, ok := v.(map[string]interface{})
		if !ok {
			// Si el valor no es un mapa, quizás es un simple value, intentar reconstruir objeto
			continue
		}

		// Intentar obtener IOA del mapa de propiedades, o de la clave del mapa superior
		var ioa int
		if valIOA, exists := props["ioa"]; exists {
			ioa = ConvertToInt(valIOA).(int)
		} else {
			ioa = ConvertToInt(k).(int)
			props["ioa"] = ioa // Asegurarse de que el IOA esté dentro del objeto para el YAML
		}

		items = append(items, item{ioa: ioa, val: props})
	}

	// Ordenar por IOA
	sort.Slice(items, func(i, j int) bool {
		return items[i].ioa < items[j].ioa
	})

	for _, it := range items {
		list = append(list, it.val)
	}

	return list
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

	err := g.Clean()
	if err != nil {
		return fmt.Errorf("failed to clean configuration path: %w", err)
	}

	// Generate master configurations
	masterIndex := 0
	for _, node := range g.scenario.Nodes {
		if IsMaster(node) {
			err := g.CraftMaster(node.Messages, masterIndex)
			if err != nil {
				return fmt.Errorf("failed to craft master %d: %w", masterIndex, err)
			}
			masterIndex++
		}
	}

	// Generate slave configurations
	slaveIndex := 0
	for _, node := range g.scenario.Nodes {
		if IsSlave(node) {
			err := g.CraftSlave(node, slaveIndex)
			if err != nil {
				return fmt.Errorf("failed to craft slave %d: %w", slaveIndex, err)
			}
			slaveIndex++
		}
	}

	return nil
}

// Helper functions

func IsMaster(node Node) bool {
	return node.Role == "master"
}

func IsSlave(node Node) bool {
	return node.Role == "slave"
}

func FilterMasters(nodes []Node) []Node {
	var masters []Node
	for _, node := range nodes {
		if IsMaster(node) {
			masters = append(masters, node)
		}
	}
	return masters
}

func FilterSlaves(nodes []Node) []Node {
	var slaves []Node
	for _, node := range nodes {
		if IsSlave(node) {
			slaves = append(slaves, node)
		}
	}
	return slaves
}

func (g *Generator) GetMasterCount() int {
	count := 0
	for _, node := range g.scenario.Nodes {
		if IsMaster(node) {
			count++
		}
	}
	return count
}

func (g *Generator) GetSlaveCount() int {
	count := 0
	for _, node := range g.scenario.Nodes {
		if IsSlave(node) {
			count++
		}
	}
	return count
}

func GenerateFromMap(scenarioMap map[string]interface{}, configPath string) error {
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

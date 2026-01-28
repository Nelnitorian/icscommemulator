package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"icscommemulator/pkg/adapter"
	"icscommemulator/pkg/config"
	"icscommemulator/pkg/docker"
	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/runner"
)

type ScenarioService struct {
	converter *adapter.Converter
	runnerSvc runner.Service
	workDir   string
}

func NewScenarioService(workDir string, runnerSvc runner.Service) *ScenarioService {
	return &ScenarioService{
		converter: adapter.NewConverter(),
		runnerSvc: runnerSvc,
		workDir:   workDir,
	}
}

type RunScenarioRequest struct {
	CytoscapeData  adapter.CytoscapeData
	SimulationTime int
	Network        *NetworkEmulation
}

type NetworkEmulation struct {
	RateLimitMBps     float64
	PacketLossPercent float64
}

type RunScenarioResult struct {
	Message        string
	SimulationTime int
	FilePath       string
}

func (s *ScenarioService) RunScenario(req RunScenarioRequest) (*RunScenarioResult, error) {
	logger.Info("Starting scenario execution")

	logs := adapter.ValidateCytoscapeScenario(req.CytoscapeData, adapter.ERROR)
	if len(logs) > 0 {
		return nil, fmt.Errorf("scenario validation failed: %v", logs)
	}

	simplified, err := s.converter.CytoscapeToSimplified(req.CytoscapeData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert scenario: %w", err)
	}

	logger.Debug("Converted scenario to simplified format")

	attackLabels, err := s.applyAttacks(simplified, req.CytoscapeData.Protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to apply attacks: %w", err)
	}

	// Use a unique temp dir to avoid collisions across concurrent runs.
	dockerComposePath := "docker-compose.yml"
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("ICSCommEmulator-%d-%d", os.Getuid(), time.Now().UnixNano()))
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp config directory: %w", err)
	}

	if err := s.generateConfigurations(simplified, req.CytoscapeData.Protocol, dockerComposePath, configPath); err != nil {
		return nil, fmt.Errorf("failed to generate configurations: %w", err)
	}

	if err := s.verifyConfigFiles(configPath); err != nil {
		return nil, fmt.Errorf("configuration verification failed: %w", err)
	}

	pcapFilename := fmt.Sprintf("scenario_%d.pcap", time.Now().Unix())
	var networkEmulation *runner.NetworkEmulation
	if req.Network != nil {
		networkEmulation = &runner.NetworkEmulation{
			RateLimitMBps:     req.Network.RateLimitMBps,
			PacketLossPercent: req.Network.PacketLossPercent,
		}
	}

	filePath, err := s.runnerSvc.Start(dockerComposePath, req.SimulationTime, pcapFilename, configPath, networkEmulation)
	if err != nil {
		return nil, fmt.Errorf("failed to start scenario: %w", err)
	}

	if len(attackLabels) > 0 {
		if err := writeAttackLabels(filePath, req.CytoscapeData.Protocol, attackLabels); err != nil {
			logger.Warning("Failed to write attack labels: %v", err)
		}
	}

	logger.Info("Scenario started successfully")

	return &RunScenarioResult{
		Message:        "Scenario running",
		SimulationTime: req.SimulationTime,
		FilePath:       filePath,
	}, nil
}

func (s *ScenarioService) generateConfigurations(simplified map[string]interface{}, protocol, dockerComposePath, configPath string) error {
	logger.Debug("Generating configurations for protocol: %s", protocol)

	var scenarioConfig config.Scenario
	scenarioConfig.Protocol = protocol

	if nodesInterface, ok := simplified["nodes"]; ok {
		nodesData, err := yaml.Marshal(nodesInterface)
		if err != nil {
			return fmt.Errorf("failed to marshal nodes: %w", err)
		}

		if err := yaml.Unmarshal(nodesData, &scenarioConfig.Nodes); err != nil {
			return fmt.Errorf("failed to unmarshal nodes: %w", err)
		}
	}

	logger.Debug("Scenario config created with %d nodes", len(scenarioConfig.Nodes))

	for i, node := range scenarioConfig.Nodes {
		logger.Debug("Node %d: role=%s, messages=%d", i, node.Role, len(node.Messages))
	}

	configGen := config.NewGenerator(scenarioConfig, configPath)

	logger.Info("Calling config.Generator.Generate()...")
	if err := configGen.Generate(); err != nil {
		return fmt.Errorf("failed to generate protocol configs: %w", err)
	}

	logger.Info("Protocol configuration files generated successfully")

	scenarioPath := filepath.Join(configPath, "scenario.yaml")
	yamlData, err := yaml.Marshal(simplified)
	if err != nil {
		return fmt.Errorf("failed to marshal scenario: %w", err)
	}

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(scenarioPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write scenario file: %w", err)
	}

	logger.Debug("Created scenario.yaml")

	if err := docker.GenerateFromScenario(scenarioPath, configPath, dockerComposePath, protocol); err != nil {
		return fmt.Errorf("failed to generate docker-compose: %w", err)
	}

	logger.Info("Docker Compose generated: %s", dockerComposePath)
	return nil
}

func (s *ScenarioService) verifyConfigFiles(configPath string) error {
	mastersDir := filepath.Join(configPath, "masters")
	slavesDir := filepath.Join(configPath, "slaves")

	if _, err := os.Stat(mastersDir); os.IsNotExist(err) {
		return fmt.Errorf("masters directory not created: %s", mastersDir)
	}

	if _, err := os.Stat(slavesDir); os.IsNotExist(err) {
		return fmt.Errorf("slaves directory not created: %s", slavesDir)
	}

	masterFiles, _ := os.ReadDir(mastersDir)
	slaveFiles, _ := os.ReadDir(slavesDir)

	logger.Debug("Found %d master dirs, %d slave dirs", len(masterFiles), len(slaveFiles))

	if len(masterFiles) > 0 {
		firstMaster := filepath.Join(mastersDir, masterFiles[0].Name(), "master.yaml")
		if _, err := os.Stat(firstMaster); err != nil {
			return fmt.Errorf("master.yaml not found: %w", err)
		}
		logger.Debug("Verified master.yaml exists")
	}

	if len(slaveFiles) > 0 {
		firstSlave := filepath.Join(slavesDir, slaveFiles[0].Name(), "slave.yaml")
		if _, err := os.Stat(firstSlave); err != nil {
			return fmt.Errorf("slave.yaml not found: %w", err)
		}
		logger.Debug("Verified slave.yaml exists")
	}

	return nil
}

func (s *ScenarioService) StopScenario() error {
	return s.runnerSvc.Stop()
}

func (s *ScenarioService) GetScenarioStatus() runner.Status {
	return s.runnerSvc.GetStatus()
}

type AttackLabel struct {
	AttackID    string                 `json:"attack_id"`
	TechniqueID string                 `json:"technique_id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Protocol    string                 `json:"protocol"`
	SourceNode  string                 `json:"source_node"`
	TargetNode  string                 `json:"target_node"`
	TargetIP    string                 `json:"target_ip,omitempty"`
	TargetPort  int                    `json:"target_port,omitempty"`
	StartTime   int                    `json:"start_time"`
	Interval    int                    `json:"interval,omitempty"`
	Count       int                    `json:"count,omitempty"`
	Message     map[string]interface{} `json:"message"`
}

type attackLabelFile struct {
	SchemaVersion string        `json:"schema_version"`
	GeneratedAt   string        `json:"generated_at"`
	Protocol      string        `json:"protocol"`
	PcapPath      string        `json:"pcap_path"`
	Attacks       []AttackLabel `json:"attacks"`
}

type AttackCatalog struct {
	SchemaVersion string             `json:"schema_version"`
	Attacks       []AttackDefinition `json:"attacks"`
}

type AttackDefinition struct {
	ID               string                    `json:"id"`
	Protocol         string                    `json:"protocol"`
	TechniqueID      string                    `json:"technique_id,omitempty"`
	Name             string                    `json:"name,omitempty"`
	Description      string                    `json:"description,omitempty"`
	ScheduleDefaults AttackScheduleDefaults    `json:"schedule_defaults,omitempty"`
	Parameters       []AttackParameterMetadata `json:"parameters,omitempty"`
	MessageTemplate  map[string]interface{}    `json:"message_template,omitempty"`
}

type AttackScheduleDefaults struct {
	StartTime int `json:"start_time,omitempty"`
	Interval  int `json:"interval,omitempty"`
	Count     int `json:"count,omitempty"`
}

type AttackParameterMetadata struct {
	Name        string      `json:"name"`
	Type        string      `json:"type,omitempty"`
	Label       string      `json:"label,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Placeholder string      `json:"placeholder,omitempty"`
}

func writeAttackLabels(pcapPath, protocol string, labels []AttackLabel) error {
	output := attackLabelFile{
		SchemaVersion: "icscommemulator.attack-labels.v1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Protocol:      protocol,
		PcapPath:      pcapPath,
		Attacks:       labels,
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}
	labelPath := pcapPath + ".labels.json"
	if err := os.WriteFile(labelPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write labels file: %w", err)
	}
	return nil
}

func (s *ScenarioService) applyAttacks(simplified map[string]interface{}, protocol string) ([]AttackLabel, error) {
	catalog, err := loadAttackCatalog()
	if err != nil {
		logger.Warning("Failed to load attack catalog: %v", err)
	}
	definitions := map[string]AttackDefinition{}
	for _, def := range catalog.Attacks {
		if def.ID == "" || def.Protocol == "" {
			continue
		}
		definitions[fmt.Sprintf("%s|%s", def.Protocol, def.ID)] = def
	}

	var nodeMaps []map[string]interface{}
	switch nodesTyped := simplified["nodes"].(type) {
	case []map[string]interface{}:
		nodeMaps = nodesTyped
	case []interface{}:
		for _, raw := range nodesTyped {
			nodeMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			nodeMaps = append(nodeMaps, nodeMap)
		}
	default:
		return nil, nil
	}

	nodeByID := make(map[string]map[string]interface{})
	var slaveIDs []string
	for _, node := range nodeMaps {
		id := getString(node["id"])
		if id == "" {
			continue
		}
		nodeByID[id] = node
		if getString(node["role"]) == "slave" {
			slaveIDs = append(slaveIDs, id)
		}
	}

	var labels []AttackLabel
	for _, node := range nodeMaps {
		if getString(node["role"]) != "master" {
			continue
		}

		attacks := decodeAttacks(node["attacks"])
		if len(attacks) == 0 {
			continue
		}

		messages := decodeMessages(node["messages"])
		for _, attack := range attacks {
			if !attack.Enabled {
				continue
			}
			def, ok := definitions[fmt.Sprintf("%s|%s", protocol, attack.ID)]
			if !ok {
				logger.Warning("Attack definition not found for %s (%s)", attack.ID, protocol)
				continue
			}
			targetID := attack.TargetID
			if targetID == "" && len(slaveIDs) == 1 {
				targetID = slaveIDs[0]
			}
			target := nodeByID[targetID]
			if targetID == "" || target == nil {
				logger.Warning("Skipping attack %s: target not set", attack.ID)
				continue
			}

			newMessages, attackLabel := buildAttackMessages(attack, def, protocol, node, target)
			if len(newMessages) == 0 {
				continue
			}
			messages = append(messages, newMessages...)
			if attackLabel.AttackID != "" {
				labels = append(labels, attackLabel)
			}
		}

		if len(messages) > 0 {
			node["messages"] = messages
		}
	}

	simplified["nodes"] = nodeMaps
	return labels, nil
}

func decodeAttacks(raw interface{}) []adapter.AttackConfig {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var attacks []adapter.AttackConfig
	if err := json.Unmarshal(data, &attacks); err != nil {
		return nil
	}
	return attacks
}

func decodeMessages(raw interface{}) []map[string]interface{} {
	if raw == nil {
		return []map[string]interface{}{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return []map[string]interface{}{}
	}
	var messages []map[string]interface{}
	if err := json.Unmarshal(data, &messages); err != nil {
		return []map[string]interface{}{}
	}
	return messages
}

func loadAttackCatalog() (AttackCatalog, error) {
	path := os.Getenv("ICS_ATTACKS_CATALOG_PATH")
	if path == "" {
		path = "web/static/attacks/attacks.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AttackCatalog{}, err
	}
	var catalog AttackCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return AttackCatalog{}, err
	}
	return catalog, nil
}

func buildAttackMessages(
	attack adapter.AttackConfig,
	definition AttackDefinition,
	protocol string,
	source map[string]interface{},
	target map[string]interface{},
) ([]map[string]interface{}, AttackLabel) {
	start, interval, count := resolveAttackSchedule(attack, definition)

	targetIP := getString(target["ip"])
	targetPort := getInt(target["port"], defaultPortFor(protocol))
	targetID := getString(target["id"])
	sourceID := getString(source["id"])
	masterID := getInt(source["master_id"], getInt(target["master_id"], 0))
	outstationID := getInt(target["outstation_id"], 0)
	commonAddress := getInt(target["common_address"], 0)
	slaveID := getInt(target["slave_id"], 0)

	baseMessage := map[string]interface{}{
		"recurrent":      false,
		"interval":       0,
		"ip":             targetIP,
		"port":           targetPort,
		"master_id":      masterID,
		"outstation_id":  outstationID,
		"common_address": commonAddress,
		"slave_id":       slaveID,
	}

	if len(definition.MessageTemplate) == 0 {
		switch protocol {
		case "modbus":
			buildModbusAttack(attack, target, baseMessage)
		case "dnp3":
			buildDNP3Attack(attack, target, baseMessage)
		case "iec104":
			buildIEC104Attack(attack, target, baseMessage)
		default:
			return nil, AttackLabel{}
		}
	}

	templateMessage := renderMessageTemplate(definition.MessageTemplate, resolveAttackParameters(attack, definition))
	mergedMessage := mergeMessageTemplate(baseMessage, templateMessage)

	var messages []map[string]interface{}
	for i := 0; i < count; i++ {
		msg := copyMap(mergedMessage)
		msg["timestamp"] = start + (i * max(1, interval))
		messages = append(messages, msg)
	}

	label := AttackLabel{
		AttackID:    attack.ID,
		TechniqueID: pickFirst(attack.TechniqueID, definition.TechniqueID),
		Name:        pickFirst(attack.Name, definition.Name),
		Protocol:    protocol,
		SourceNode:  sourceID,
		TargetNode:  targetID,
		TargetIP:    targetIP,
		TargetPort:  targetPort,
		StartTime:   start,
		Interval:    interval,
		Count:       count,
		Message:     mergedMessage,
	}

	return messages, label
}

func buildModbusAttack(attack adapter.AttackConfig, target map[string]interface{}, message map[string]interface{}) {
	address := getParamInt(attack.Parameters, "address", 0)
	value := getParamInt(attack.Parameters, "value", 1)
	values := getParamValues(attack.Parameters, "values")
	countRegisters := getParamInt(attack.Parameters, "count_registers", 10)
	slaveID := getInt(target["slave_id"], 1)

	message["slave_id"] = slaveID

	switch attack.ID {
	case "modbus_write_single_coil":
		message["function_code"] = 5
		message["start_address"] = address
		message["values"] = []interface{}{value}
		message["count"] = 1
	case "modbus_write_multiple_registers":
		message["function_code"] = 16
		message["start_address"] = address
		message["values"] = values
		message["count"] = len(values)
	case "modbus_write_multiple_coils":
		message["function_code"] = 15
		message["start_address"] = address
		message["values"] = values
		message["count"] = len(values)
	case "modbus_poll_flood":
		message["function_code"] = 3
		message["start_address"] = address
		message["count"] = countRegisters
	default:
		message["function_code"] = 6
		message["start_address"] = address
		message["values"] = []interface{}{value}
		message["count"] = 1
	}
}

func buildDNP3Attack(attack adapter.AttackConfig, target map[string]interface{}, message map[string]interface{}) {
	index := getParamInt(attack.Parameters, "index", 0)
	value := getParamString(attack.Parameters, "value", "1")
	masterID := getInt(target["master_id"], 2)
	outstationID := getInt(target["outstation_id"], 1)

	message["master_id"] = masterID
	message["outstation_id"] = outstationID

	switch attack.ID {
	case "dnp3_send_analog_command_float32":
		message["operation_type"] = "send_analog_command_float32"
		message["index"] = index
		message["value"] = value
	case "dnp3_send_analog_command_int32":
		message["operation_type"] = "send_analog_command_int32"
		message["index"] = index
		message["value"] = value
	case "dnp3_poll_group_variation":
		message["operation_type"] = "poll_group_variation"
		message["group"] = getParamInt(attack.Parameters, "group", 30)
		message["variation"] = getParamInt(attack.Parameters, "variation", 6)
	case "dnp3_poll_group_variation_index":
		message["operation_type"] = "poll_group_variation_index"
		message["group"] = getParamInt(attack.Parameters, "group", 30)
		message["variation"] = getParamInt(attack.Parameters, "variation", 6)
		message["index"] = index
	case "dnp3_send_analog_command":
		message["operation_type"] = "send_analog_command_int16"
		message["index"] = index
		message["value"] = value
	case "dnp3_poll_flood":
		message["operation_type"] = "poll_all"
	default:
		message["operation_type"] = "send_binary_command"
		message["index"] = index
		message["value"] = value
	}
}

func buildIEC104Attack(attack adapter.AttackConfig, target map[string]interface{}, message map[string]interface{}) {
	ioa := getParamInt(attack.Parameters, "ioa", 1)
	value := getParamString(attack.Parameters, "value", "on")
	commonAddress := getInt(target["common_address"], 1)

	message["common_address"] = commonAddress

	switch attack.ID {
	case "iec104_read_command":
		message["type_id"] = 102
		message["ioa"] = ioa
		message["cot"] = 6
		message["value"] = ""
	case "iec104_clock_sync":
		message["type_id"] = 103
		message["ioa"] = 0
		message["cot"] = 6
		message["value"] = ""
	case "iec104_double_command":
		message["type_id"] = 46
		message["ioa"] = ioa
		message["cot"] = 6
		message["value"] = value
	case "iec104_setpoint_scaled":
		message["type_id"] = 49
		message["ioa"] = ioa
		message["cot"] = 6
		message["value"] = value
	case "iec104_setpoint_command":
		message["type_id"] = 50
		message["ioa"] = ioa
		message["cot"] = 6
		message["value"] = value
	case "iec104_interrogation_flood":
		message["type_id"] = 100
		message["ioa"] = 0
		message["cot"] = 6
		message["value"] = ""
	default:
		message["type_id"] = 45
		message["ioa"] = ioa
		message["cot"] = 6
		message["value"] = value
	}
}

func defaultPortFor(protocol string) int {
	switch protocol {
	case "modbus":
		return 502
	case "dnp3":
		return 20000
	case "iec104":
		return 2404
	default:
		return 0
	}
}

func copyMap(input map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(input))
	for k, v := range input {
		output[k] = v
	}
	return output
}

func getString(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func getInt(value interface{}, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if v == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getParamInt(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	return getInt(params[key], fallback)
}

func getParamString(params map[string]interface{}, key, fallback string) string {
	if params == nil {
		return fallback
	}
	if v, ok := params[key].(string); ok {
		if v != "" {
			return v
		}
	}
	return fallback
}

func getParamValues(params map[string]interface{}, key string) []interface{} {
	if params == nil {
		return []interface{}{1}
	}
	raw := params[key]
	switch v := raw.(type) {
	case []interface{}:
		if len(v) > 0 {
			return v
		}
	case []string:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case string:
		if v == "" {
			return []interface{}{1}
		}
		parts := strings.Split(v, ",")
		out := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if num, err := strconv.Atoi(part); err == nil {
				out = append(out, num)
			} else {
				out = append(out, part)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []interface{}{1}
}

func resolveAttackSchedule(attack adapter.AttackConfig, def AttackDefinition) (int, int, int) {
	start := attack.StartTime
	interval := attack.Interval
	count := attack.Count
	if start == 0 && def.ScheduleDefaults.StartTime > 0 {
		start = def.ScheduleDefaults.StartTime
	}
	if interval == 0 && def.ScheduleDefaults.Interval > 0 {
		interval = def.ScheduleDefaults.Interval
	}
	if count <= 0 {
		if def.ScheduleDefaults.Count > 0 {
			count = def.ScheduleDefaults.Count
		} else {
			count = 1
		}
	}
	return start, interval, count
}

func resolveAttackParameters(attack adapter.AttackConfig, def AttackDefinition) map[string]interface{} {
	params := make(map[string]interface{})
	for _, meta := range def.Parameters {
		raw := interface{}(nil)
		if attack.Parameters != nil {
			raw = attack.Parameters[meta.Name]
		}
		if raw == nil {
			raw = meta.Default
		}
		params[meta.Name] = parseParamValue(meta.Type, raw)
	}

	for key, value := range attack.Parameters {
		if _, exists := params[key]; !exists {
			params[key] = value
		}
	}

	if values, ok := params["values"]; ok {
		params["values_count"] = lenParamList(values)
	}

	return params
}

func parseParamValue(paramType string, raw interface{}) interface{} {
	switch strings.ToLower(paramType) {
	case "int":
		return getInt(raw, 0)
	case "float":
		return getFloat(raw, 0)
	case "bool":
		return getBool(raw, false)
	case "list_int":
		return parseList(raw, true)
	case "list_float":
		return parseListFloat(raw)
	case "list_string":
		return parseList(raw, false)
	case "string":
		return fmt.Sprintf("%v", raw)
	default:
		return raw
	}
}

func parseList(raw interface{}, numbers bool) []interface{} {
	switch v := raw.(type) {
	case []interface{}:
		return v
	case []string:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			if item == "" {
				continue
			}
			if numbers {
				out = append(out, getInt(item, 0))
			} else {
				out = append(out, item)
			}
		}
		return out
	case string:
		if v == "" {
			return []interface{}{}
		}
		parts := strings.Split(v, ",")
		out := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if numbers {
				out = append(out, getInt(part, 0))
			} else {
				out = append(out, part)
			}
		}
		return out
	default:
		if raw == nil {
			return []interface{}{}
		}
		return []interface{}{raw}
	}
}

func parseListFloat(raw interface{}) []interface{} {
	switch v := raw.(type) {
	case []interface{}:
		return v
	case []string:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			if item == "" {
				continue
			}
			out = append(out, getFloat(item, 0))
		}
		return out
	case string:
		if v == "" {
			return []interface{}{}
		}
		parts := strings.Split(v, ",")
		out := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, getFloat(part, 0))
		}
		return out
	default:
		if raw == nil {
			return []interface{}{}
		}
		return []interface{}{raw}
	}
}

func renderMessageTemplate(template map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	if len(template) == 0 {
		return map[string]interface{}{}
	}
	rendered := make(map[string]interface{}, len(template))
	for key, value := range template {
		rendered[key] = renderTemplateValue(value, params)
	}
	return rendered
}

func renderTemplateValue(value interface{}, params map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return renderTemplateString(v, params)
	case map[string]interface{}:
		return renderMessageTemplate(v, params)
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			out = append(out, renderTemplateValue(item, params))
		}
		return out
	default:
		return value
	}
}

func renderTemplateString(value string, params map[string]interface{}) interface{} {
	expression := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
	matches := expression.FindAllStringSubmatch(value, -1)
	if len(matches) == 1 && strings.TrimSpace(value) == matches[0][0] {
		if val, ok := params[matches[0][1]]; ok {
			return val
		}
		return value
	}
	result := expression.ReplaceAllStringFunc(value, func(match string) string {
		sub := expression.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		if val, ok := params[sub[1]]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
	return result
}

func mergeMessageTemplate(base, overlay map[string]interface{}) map[string]interface{} {
	merged := copyMap(base)
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func lenParamList(value interface{}) int {
	switch v := value.(type) {
	case []interface{}:
		return len(v)
	case []string:
		return len(v)
	default:
		return 0
	}
}

func getFloat(value interface{}, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || strings.EqualFold(v, "1") || strings.EqualFold(v, "yes")
	case int:
		return v != 0
	default:
		return fallback
	}
}

func pickFirst(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// cleanDirectory removes the work directory with guardrails.
func cleanDirectory(path string) error {
	if path == "" || path == "/" || path == "/tmp" {
		return fmt.Errorf("invalid path: %s", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(path)
}

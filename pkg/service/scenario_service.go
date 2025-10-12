package service

import (
	"fmt"
	"os"
	"path/filepath"
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
}

type RunScenarioResult struct {
	Message        string
	SimulationTime int
	FilePath       string
}

func (s *ScenarioService) RunScenario(req RunScenarioRequest) (*RunScenarioResult, error) {
	logger.Info("Starting scenario execution")

	// 0. Limpiar ANTES de empezar (aquí sí tiene sentido)
	if err := cleanDirectory(s.workDir); err != nil {
		logger.Warning("Failed to clean work directory before start: %v", err)
	}

	// 1. Validar
	logs := adapter.ValidateCytoscapeScenario(req.CytoscapeData, adapter.ERROR)
	if len(logs) > 0 {
		return nil, fmt.Errorf("scenario validation failed: %v", logs)
	}

	// 2. Convertir
	simplified, err := s.converter.CytoscapeToSimplified(req.CytoscapeData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert scenario: %w", err)
	}

	logger.Debug("Converted scenario to simplified format")

	// 3. Generar configs
	dockerComposePath := "docker-compose.yml"
	configPath := s.workDir

	if err := s.generateConfigurations(simplified, req.CytoscapeData.Protocol, dockerComposePath, configPath); err != nil {
		return nil, fmt.Errorf("failed to generate configurations: %w", err)
	}

	// 4. Verificar archivos
	if err := s.verifyConfigFiles(configPath); err != nil {
		return nil, fmt.Errorf("configuration verification failed: %w", err)
	}

	// 5. Ejecutar escenario (NO limpiar aquí - el runner lo hará al finalizar)
	pcapFilename := fmt.Sprintf("scenario_%d.pcap", time.Now().Unix())
	filePath, err := s.runnerSvc.Start(dockerComposePath, req.SimulationTime, pcapFilename, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to start scenario: %w", err)
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

	// Crear config.Scenario
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

	// Generar masters/slaves
	configGen := config.NewGenerator(scenarioConfig, configPath)
	
	logger.Info("Calling config.Generator.Generate()...")
	if err := configGen.Generate(); err != nil {
		return fmt.Errorf("failed to generate protocol configs: %w", err)
	}

	logger.Info("Protocol configuration files generated successfully")

	// scenario.yaml temporal
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

	// Docker Compose
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
		firstMaster := filepath.Join(mastersDir, masterFiles[0].Name(), "master.csv")
		if _, err := os.Stat(firstMaster); err != nil {
			return fmt.Errorf("master.csv not found: %w", err)
		}
		logger.Debug("Verified master.csv exists")
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

// cleanDirectory es una función helper simple
func cleanDirectory(path string) error {
	if path == "" || path == "/" || path == "/tmp" {
		return fmt.Errorf("invalid path: %s", path)
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	
	return os.RemoveAll(path)
}

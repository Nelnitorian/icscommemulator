package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"icscommemulator/pkg/logger"
)

type Service interface {
	Start(dockerComposePath string, simulationTime int, outputFile, configPath string) (string, error)
	Stop() error
	GetStatus() Status
	IsRunning() bool
}

type service struct {
	mu           sync.RWMutex
	runner       *Runner
	isRunning    bool
	currentError error
}

func NewService() Service {
	return &service{
		runner: NewRunner(),
	}
}

func (s *service) Start(dockerComposePath string, simulationTime int, outputFile, configPath string) (string, error) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return "", fmt.Errorf("a scenario is already running")
	}
	s.isRunning = true
	s.currentError = nil
	s.mu.Unlock()

	// NO LIMPIAR AQUÍ - los archivos acaban de generarse

	s.runner.Configure(dockerComposePath, simulationTime, outputFile, configPath)

	go func() {
		if err := s.runner.Run(); err != nil {
			logger.Error("Error running scenario: %v", err)
			s.mu.Lock()
			s.currentError = err
			s.mu.Unlock()
		}
		
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		
		// Limpiar DESPUÉS de que Docker termine
		time.Sleep(1 * time.Second)
		if err := cleanConfigDirectory(configPath); err != nil {
			logger.Warning("Failed to clean config directory after run: %v", err)
		}
	}()

	absPath, _ := filepath.Abs(s.runner.outputFile)
	return absPath, nil
}

func (s *service) Stop() error {
	s.mu.RLock()
	isRunning := s.isRunning
	configPath := s.runner.configPath
	s.mu.RUnlock()
	
	if !isRunning {
		return fmt.Errorf("no scenario is running")
	}

	err := s.runner.Stop()
	
	// Limpiar después de detener
	time.Sleep(1 * time.Second)
	if cleanErr := cleanConfigDirectory(configPath); cleanErr != nil {
		logger.Warning("Failed to clean after stop: %v", cleanErr)
	}
	
	return err
}

func (s *service) GetStatus() Status {
	return s.runner.Status()
}

func (s *service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

func cleanConfigDirectory(configPath string) error {
	if configPath == "" || configPath == "/" || configPath == "/tmp" {
		return fmt.Errorf("invalid config path: %s", configPath)
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Debug("Config directory does not exist: %s", configPath)
		return nil
	}
	
	logger.Info("Cleaning config directory: %s", configPath)
	
	if err := os.RemoveAll(configPath); err != nil {
		logger.Warning("Failed to remove config directory: %v", err)
		return err
	}
	
	logger.Info("Config directory cleaned successfully")
	return nil
}

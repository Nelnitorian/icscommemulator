package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tbrandon/mbserver"
	"gopkg.in/yaml.v3"
)

type RegisterConfig struct {
	Type   string      `yaml:"type"`
	Values interface{} `yaml:"values"`
}

type IdentityConfig struct {
	VendorName            string `yaml:"vendor_name"`
	ProductCode           string `yaml:"product_code"`
	VendorURL             string `yaml:"vendor_url"`
	ProductName           string `yaml:"product_name"`
	ModelName             string `yaml:"model_name"`
	MajorMinorRevision    string `yaml:"major_minor_revision"`
	UserApplicationName   string `yaml:"user_application_name"`
}

type SlaveConfig struct {
	IP                string         `yaml:"ip"`
	Port              interface{}    `yaml:"port"`
	SlaveID           interface{}    `yaml:"slave_id"`
	DiscreteInputs    RegisterConfig `yaml:"discrete_inputs"`
	Coils             RegisterConfig `yaml:"coils"`
	InputRegisters    RegisterConfig `yaml:"input_registers"`
	HoldingRegisters  RegisterConfig `yaml:"holding_registers"`
	Identity          IdentityConfig `yaml:"identity"`
}

type ModbusSlave struct {
	config   SlaveConfig
	syncFile string
	logger   *logrus.Logger
	server   *mbserver.Server
}

func NewModbusSlave(configFile, syncFile string) (*ModbusSlave, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SlaveConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &ModbusSlave{
		config:   config,
		syncFile: syncFile,
		logger:   logger,
	}, nil
}

func (s *ModbusSlave) parseValues(regConfig RegisterConfig) []uint16 {
	switch v := regConfig.Values.(type) {
	case []interface{}:
		values := make([]uint16, len(v))
		for i, val := range v {
			switch num := val.(type) {
			case int:
				values[i] = uint16(num)
			case float64:
				values[i] = uint16(num)
			case string:
				if parsed, err := strconv.Atoi(num); err == nil {
					values[i] = uint16(parsed)
				}
			}
		}
		return values
	case string:
		if v == "" {
			return []uint16{}
		}
		// Handle comma-separated values
		parts := strings.Split(v, ",")
		values := make([]uint16, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				if parsed, err := strconv.Atoi(part); err == nil {
					values = append(values, uint16(parsed))
				}
			}
		}
		return values
	default:
		return []uint16{}
	}
}

func (s *ModbusSlave) touchSyncFile() error {
	if err := os.MkdirAll(filepath.Dir(s.syncFile), 0755); err != nil {
		return err
	}
	
	file, err := os.Create(s.syncFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	s.logger.Infof("Created sync file: %s", s.syncFile)
	return nil
}

func (s *ModbusSlave) getPort() int {
	switch p := s.config.Port.(type) {
	case int:
		return p
	case string:
		if port, err := strconv.Atoi(p); err == nil {
			return port
		}
	case float64:
		return int(p)
	}
	return 502 // Default Modbus port
}

func (s *ModbusSlave) start() error {
	port := s.getPort()
	
	// Create server
	s.server = mbserver.NewServer()
	
	// Parse register configurations and set initial values
	diValues := s.parseValues(s.config.DiscreteInputs)
	coValues := s.parseValues(s.config.Coils)
	irValues := s.parseValues(s.config.InputRegisters)
	hrValues := s.parseValues(s.config.HoldingRegisters)
	
	// Set discrete inputs
	for i, val := range diValues {
		s.server.DiscreteInputs[i] = byte(val)
	}
	
	// Set coils
	for i, val := range coValues {
		s.server.Coils[i] = byte(val)
	}
	
	// Set input registers
	for i, val := range irValues {
		s.server.InputRegisters[i] = val
	}
	
	// Set holding registers
	for i, val := range hrValues {
		s.server.HoldingRegisters[i] = val
	}
	
	s.logger.Infof("Starting Modbus TCP Server on %s:%d", s.config.IP, port)
	
	// Create sync file to indicate server is running
	if err := s.touchSyncFile(); err != nil {
		return fmt.Errorf("failed to create sync file: %w", err)
	}
	
	// Start server in goroutine
	go func() {
		address := fmt.Sprintf("%s:%d", s.config.IP, port)
		if err := s.server.ListenTCP(address); err != nil {
			s.logger.WithError(err).Error("Server stopped with error")
		}
	}()
	
	// Wait a bit to ensure server started
	time.Sleep(100 * time.Millisecond)
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	s.logger.Info("Server started successfully, waiting for signals...")
	<-sigChan
	
	s.logger.Info("Received shutdown signal")
	return s.shutdown()
}

func (s *ModbusSlave) shutdown() error {
	if s.server != nil {
		s.server.Close()
	}
	
	// Remove sync file
	if err := os.Remove(s.syncFile); err != nil && !os.IsNotExist(err) {
		s.logger.WithError(err).Warn("Failed to remove sync file")
	}
	
	s.logger.Info("Server shutdown completed")
	return nil
}

func main() {
	fmt.Println("Starting ModbusSlave...")

	// Allow config file paths to be overridden via environment variables
	configFile := "slave.yaml"
	if envConfig := os.Getenv("SLAVE_CONFIG"); envConfig != "" {
		configFile = envConfig
	}
	
	syncFile := "app_running.lock"
	if envSync := os.Getenv("SYNC_FILE"); envSync != "" {
		syncFile = envSync
	}

	slave, err := NewModbusSlave(configFile, syncFile)
	if err != nil {
		log.Fatalf("Error creating slave: %v", err)
	}

	if err := slave.start(); err != nil {
		log.Fatalf("Error starting slave: %v", err)
	}
}

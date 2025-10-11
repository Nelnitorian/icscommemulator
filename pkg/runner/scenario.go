package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Status represents the current status of a running scenario
type Status struct {
	ElapsedSeconds int  `json:"elapsed_seconds"`
	TotalSeconds   int  `json:"total_seconds"`
	PcapSize       int64 `json:"pcap_size"`
	Running        bool `json:"running"`
	Error          string `json:"error,omitempty"`
}

// Config holds the configuration for a scenario run
type Config struct {
	DockerComposePath string
	SimulationTime    int
	OutputFile        string
	ConfigPath        string
}

// Runner manages scenario execution with Docker Compose and network capture
type Runner struct {
	mu sync.RWMutex
	
	// Configuration
	config *Config
	
	// State
	isRunning     bool
	startTime     *time.Time
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Processes
	tcpdumpCmd    *exec.Cmd
	
	// Paths
	filePath     string
	configPath   string
	outputFolder string
	outputFile   string
}

var (
	// Global singleton instance
	globalRunner *Runner
	globalMutex  sync.Mutex
)

// NewRunner creates a new scenario runner
func NewRunner() *Runner {
	return &Runner{
		outputFolder: "outputs",
		isRunning:    false,
	}
}

// GetGlobalRunner returns the global singleton runner instance
func GetGlobalRunner() *Runner {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	
	if globalRunner == nil {
		globalRunner = NewRunner()
	}
	return globalRunner
}

// Configure sets up the runner with the provided configuration
func (r *Runner) Configure(dockerComposePath string, simulationTime int, outputFile, configPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if configPath == "" {
		configPath = "/tmp/ICSCommEmulator"
	}
	
	r.config = &Config{
		DockerComposePath: dockerComposePath,
		SimulationTime:    simulationTime,
		OutputFile:        outputFile,
		ConfigPath:        configPath,
	}
	
	r.filePath = dockerComposePath
	r.configPath = configPath
	r.outputFile = filepath.Join(r.outputFolder, outputFile)
}

// IsRunning returns whether a scenario is currently running
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRunning
}

// GetDockerNetworkInterface extracts the network interface name from docker-compose file
func (r *Runner) GetDockerNetworkInterface() (string, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read docker-compose file: %w", err)
	}
	
	var compose struct {
		Networks map[string]interface{} `yaml:"networks"`
	}
	
	err = yaml.Unmarshal(data, &compose)
	if err != nil {
		return "", fmt.Errorf("failed to parse docker-compose file: %w", err)
	}
	
	for networkName := range compose.Networks {
		return networkName, nil
	}
	
	return "", fmt.Errorf("no networks found in docker-compose file")
}

// GetSystemInterfaceName gets the system interface name for a Docker network
func (r *Runner) GetSystemInterfaceName(dockerNetworkName string) (string, error) {

	log.Printf("Looking for Docker network: %s", dockerNetworkName)
    
    // List all networks to debug
    debugCmd := exec.Command("docker", "network", "ls")
    debugOutput, _ := debugCmd.Output()
    log.Printf("Available networks:\n%s", string(debugOutput))

    // List all networks and filter by name pattern
    cmd := exec.Command("docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", dockerNetworkName), "--format", "{{.ID}}")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to list docker networks: %w", err)
    }
    
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    if len(lines) == 0 || lines[0] == "" {
        return "", fmt.Errorf("no network found matching: %s", dockerNetworkName)
    }
    
    // Use the first matching network ID
    networkID := strings.TrimSpace(lines[0])
    if len(networkID) < 12 {
        return "", fmt.Errorf("invalid network ID: %s", networkID)
    }
    
    return fmt.Sprintf("br-%s", networkID[:12]), nil
}

// StartTcpdump starts network packet capture using tcpdump
func (r *Runner) StartTcpdump(ctx context.Context, interfaceName string) error {
	// Ensure output directory exists
	err := os.MkdirAll(r.outputFolder, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Build tcpdump command
	args := []string{
		"-i", interfaceName,
		"-w", r.outputFile,
		"-U", "-nn",
		"not udp port 5353", // filter mDNS
		"and not udp port 1900", // filter SSDP
	}
	
	r.tcpdumpCmd = exec.CommandContext(ctx, "tcpdump", args...)
	
	// Redirect stdout and stderr to discard
	r.tcpdumpCmd.Stdout = nil
	r.tcpdumpCmd.Stderr = nil
	
	log.Printf("Starting tcpdump on interface '%s', saving to '%s'", interfaceName, r.outputFile)
	
	err = r.tcpdumpCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start tcpdump: %w", err)
	}
	
	return nil
}

// LaunchDockerCompose starts the Docker Compose environment
func (r *Runner) LaunchDockerCompose() error {
	// Ensure the environment is clean before launching
	err := r.EnsureLaunchable()
	if err != nil {
		log.Printf("Warning: failed to clean environment: %v", err)
	}
	
	log.Println("Launching docker compose...")
	
	cmd := exec.Command("docker", "compose", "-f", r.filePath, "up", "--build", "-d", "--remove-orphans")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch docker compose: %w\nOutput: %s", err, string(output))
	}
	
	log.Println("Docker compose launched successfully")
	return nil
}

// EnsureLaunchable cleans up any existing network to ensure clean startup
func (r *Runner) EnsureLaunchable() error {
	networkName, err := r.GetDockerNetworkInterface()
	if err != nil {
		// If we can't get the network name, it's probably fine to continue
		return nil
	}
	
	// Try to remove existing network (ignore errors as it might not exist)
	cmd := exec.Command("docker", "network", "rm", networkName)
	cmd.Run() // Ignore output and errors
	
	return nil
}

// StopDockerCompose stops the Docker Compose environment
func (r *Runner) StopDockerCompose() error {
	log.Println("Stopping docker compose...")
	
	cmd := exec.Command("docker", "compose", "-f", r.filePath, "down", "--timeout", "3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop docker compose: %w\nOutput: %s", err, string(output))
	}
	
	log.Println("Docker compose stopped.")
	return nil
}

// StopTcpdump stops the tcpdump process
func (r *Runner) StopTcpdump() error {
	if r.tcpdumpCmd != nil && r.tcpdumpCmd.Process != nil {
		log.Println("Stopping tcpdump...")
		err := r.tcpdumpCmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop tcpdump: %w", err)
		}
		
		// Wait for the process to finish
		r.tcpdumpCmd.Wait()
		log.Println("Tcpdump stopped.")
	} else {
		log.Println("Tcpdump process not found.")
	}
	return nil
}

// CleanConfigFolder removes the configuration directory
func (r *Runner) CleanConfigFolder() error {
	if _, err := os.Stat(r.configPath); err == nil {
		return os.RemoveAll(r.configPath)
	}
	return nil
}

// Run executes the complete scenario
func (r *Runner) Run() error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("another scenario is already running")
	}
	r.isRunning = true
	now := time.Now()
	r.startTime = &now
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.mu.Unlock()
	
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.startTime = nil
		if r.cancel != nil {
			r.cancel()
		}
		r.mu.Unlock()
	}()
	
	// Launch Docker Compose
	err := r.LaunchDockerCompose()
	if err != nil {
		r.StopDockerCompose()
		return fmt.Errorf("failed to launch docker compose: %w", err)
	}
	
	defer func() {
		r.StopDockerCompose()
		r.CleanConfigFolder()
	}()
	
	// Get network interface
	networkName, err := r.GetDockerNetworkInterface()
	if err != nil {
		return fmt.Errorf("failed to get docker network interface: %w", err)
	}
	
	systemInterface, err := r.GetSystemInterfaceName(networkName)
	if err != nil {
		return fmt.Errorf("failed to get system interface name: %w", err)
	}
	
	// Start tcpdump
	err = r.StartTcpdump(r.ctx, systemInterface)
	if err != nil {
		return fmt.Errorf("failed to start tcpdump: %w", err)
	}
	
	defer r.StopTcpdump()
	
	// Wait for simulation time + 1 second
	select {
	case <-time.After(time.Duration(r.config.SimulationTime+1) * time.Second):
		log.Println("Simulation completed")
	case <-r.ctx.Done():
		log.Println("Simulation cancelled")
	}
	
	log.Println("Stopping network traffic capture...")
	return nil
}

// Status returns the current status of the scenario
func (r *Runner) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.startTime == nil || !r.isRunning {
		return Status{
			Error: "Simulation not started.",
		}
	}
	
	elapsed := time.Since(*r.startTime)
	elapsedSeconds := int(elapsed.Seconds())
	
	var pcapSize int64
	if info, err := os.Stat(r.outputFile); err == nil {
		pcapSize = info.Size()
	}
	
	return Status{
		ElapsedSeconds: elapsedSeconds,
		TotalSeconds:   r.config.SimulationTime,
		PcapSize:       pcapSize,
		Running:        r.isRunning,
	}
}

// Stop forcefully stops the current scenario
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.isRunning {
		return fmt.Errorf("no scenario is running")
	}
	
	if r.cancel != nil {
		r.cancel()
	}
	
	// Stop processes
	if err := r.StopTcpdump(); err != nil {
		log.Printf("Error stopping tcpdump: %v", err)
	}
	
	if err := r.StopDockerCompose(); err != nil {
		log.Printf("Error stopping docker compose: %v", err)
	}
	
	return nil
}

// Package-level functions for convenience (equivalent to Python's global functions)

// Start starts a new scenario with the given parameters
func Start(dockerComposePath string, simulationTime int, outputFile, configPath string) (string, error) {
	runner := GetGlobalRunner()
	
	if runner.IsRunning() {
		return "", fmt.Errorf("a scenario is already running")
	}
	
	runner.Configure(dockerComposePath, simulationTime, outputFile, configPath)
	
	// Start in goroutine (equivalent to Python's threading)
	go func() {
		if err := runner.Run(); err != nil {
			log.Printf("Error running scenario: %v", err)
		}
	}()
	
	// Return absolute path to output file
	absPath, err := filepath.Abs(runner.outputFile)
	if err != nil {
		return runner.outputFile, nil // Return relative path if absolute fails
	}
	
	return absPath, nil
}

// Stop stops the currently running scenario
func Stop() error {
	runner := GetGlobalRunner()
	return runner.Stop()
}

// GetStatus returns the status of the currently running scenario
func GetStatus() Status {
	runner := GetGlobalRunner()
	return runner.Status()
}

// Example usage function (equivalent to the Python __main__ block)
func ExampleUsage() error {
	outputPath, err := Start("docker-compose.yml", 10, "output.pcap", "")
	if err != nil {
		return err
	}
	
	log.Printf("Scenario started, output will be saved to: %s", outputPath)
	return nil
}

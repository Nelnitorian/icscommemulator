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

type Status struct {
	ElapsedSeconds int    `json:"elapsed_seconds"`
	TotalSeconds   int    `json:"total_seconds"`
	PcapSize       int64  `json:"pcap_size"`
	Running        bool   `json:"running"`
	Error          string `json:"error,omitempty"`
}

type Config struct {
	DockerComposePath string
	SimulationTime    int
	OutputFile        string
	ConfigPath        string
}

type Runner struct {
	mu sync.RWMutex

	config *Config

	isRunning bool
	startTime *time.Time
	ctx       context.Context
	cancel    context.CancelFunc

	tcpdumpCmd      *exec.Cmd
	networkPrepared bool

	filePath     string
	configPath   string
	outputFolder string
	outputFile   string
}

var (
	globalRunner *Runner
	globalMutex  sync.Mutex
)

func NewRunner() *Runner {
	return &Runner{
		outputFolder: "outputs",
		isRunning:    false,
	}
}

func GetGlobalRunner() *Runner {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalRunner == nil {
		globalRunner = NewRunner()
	}
	return globalRunner
}

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

func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRunning
}

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

func (r *Runner) GetDockerNetworkConfig() (string, string, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read docker-compose file: %w", err)
	}

	type ipamConfig struct {
		Config []struct {
			Subnet string `yaml:"subnet"`
		} `yaml:"config"`
	}
	type networkConfig struct {
		IPAM ipamConfig `yaml:"ipam"`
	}
	var compose struct {
		Networks map[string]networkConfig `yaml:"networks"`
	}

	if err := yaml.Unmarshal(data, &compose); err != nil {
		return "", "", fmt.Errorf("failed to parse docker-compose file: %w", err)
	}

	for name, cfg := range compose.Networks {
		subnet := ""
		if len(cfg.IPAM.Config) > 0 {
			subnet = cfg.IPAM.Config[0].Subnet
		}
		return name, subnet, nil
	}

	return "", "", fmt.Errorf("no networks found in docker-compose file")
}

func (r *Runner) PrepareNetwork() (string, error) {
	networkName, subnet, err := r.GetDockerNetworkConfig()
	if err != nil {
		return "", err
	}
	if subnet == "" {
		return "", fmt.Errorf("no subnet found for network %s", networkName)
	}

	if err := exec.Command("docker", "network", "inspect", networkName).Run(); err != nil {
		cmd := exec.Command("docker", "network", "create", "--driver", "bridge", "--subnet", subnet, networkName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to create docker network: %w\nOutput: %s", err, string(output))
		}
	}

	r.networkPrepared = true
	return r.GetSystemInterfaceName(networkName)
}

func (r *Runner) GetSystemInterfaceName(dockerNetworkName string) (string, error) {

	log.Printf("Looking for Docker network: %s", dockerNetworkName)

	debugCmd := exec.Command("docker", "network", "ls")
	debugOutput, _ := debugCmd.Output()
	log.Printf("Available networks:\n%s", string(debugOutput))

	cmd := exec.Command("docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", dockerNetworkName), "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list docker networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no network found matching: %s", dockerNetworkName)
	}

	networkID := strings.TrimSpace(lines[0])
	if len(networkID) < 12 {
		return "", fmt.Errorf("invalid network ID: %s", networkID)
	}

	return fmt.Sprintf("br-%s", networkID[:12]), nil
}

func (r *Runner) StartTcpdump(ctx context.Context, interfaceName string) error {
	err := os.MkdirAll(r.outputFolder, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filter := buildTcpdumpFilter()

	args := []string{
		"-i", interfaceName,
		"-w", r.outputFile,
		"-U",
		"-nn",
		filter,
	}

	r.tcpdumpCmd = exec.CommandContext(ctx, "tcpdump", args...)
	r.tcpdumpCmd.Stdout = nil
	r.tcpdumpCmd.Stderr = nil

	log.Printf("Starting tcpdump on interface '%s', saving to '%s'", interfaceName, r.outputFile)
	log.Printf("Filter: %s", filter)

	err = r.tcpdumpCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start tcpdump: %w", err)
	}

	return nil
}

func buildTcpdumpFilter() string {
	// Capture ICS traffic plus ARP/ICMP and drop common background noise.
	icsProtocolPorts := "(tcp port 502 or tcp port 20000 or tcp port 2404 or arp or icmp)"

	excludeFilters := []string{
		"not udp port 5353",
		"not udp port 1900",
		"not udp port 5355",
		"not udp port 137",
		"not udp port 138",
		"not tcp port 139",
		"not tcp port 445",
		"not udp port 546",
		"not udp port 547",
		"not ip6",
		"not igmp",
	}

	filter := icsProtocolPorts
	for _, exclude := range excludeFilters {
		filter += " and " + exclude
	}

	return filter
}

func (r *Runner) LaunchDockerCompose() error {
	if !r.networkPrepared {
		err := r.EnsureLaunchable()
		if err != nil {
			log.Printf("Warning: failed to clean environment: %v", err)
		}
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

func (r *Runner) EnsureLaunchable() error {
	networkName, err := r.GetDockerNetworkInterface()
	if err != nil {
		return nil
	}

	cmd := exec.Command("docker", "network", "rm", networkName)
	cmd.Run() // Ignore output and errors

	return nil
}

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

func (r *Runner) StopTcpdump() error {
	if r.tcpdumpCmd != nil && r.tcpdumpCmd.Process != nil {
		log.Println("Stopping tcpdump...")
		err := r.tcpdumpCmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop tcpdump: %w", err)
		}

		r.tcpdumpCmd.Wait()
		log.Println("Tcpdump stopped.")
	} else {
		log.Println("Tcpdump process not found.")
	}
	return nil
}

func (r *Runner) CleanConfigFolder() error {
	if _, err := os.Stat(r.configPath); err == nil {
		return os.RemoveAll(r.configPath)
	}
	return nil
}

func (r *Runner) Run() error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("another scenario is already running")
	}
	r.isRunning = true
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

	systemInterface, err := r.PrepareNetwork()
	prepared := err == nil
	if err != nil {
		log.Printf("Warning: failed to prepare network early capture: %v", err)
	}

	if prepared {
		if err := r.StartTcpdump(r.ctx, systemInterface); err != nil {
			return fmt.Errorf("failed to start tcpdump: %w", err)
		}
	}

	err = r.LaunchDockerCompose()
	if err != nil {
		r.StopDockerCompose()
		return fmt.Errorf("failed to launch docker compose: %w", err)
	}

	defer func() {
		r.StopDockerCompose()
		r.CleanConfigFolder()
	}()

	if !prepared {
		networkName, err := r.GetDockerNetworkInterface()
		if err != nil {
			return fmt.Errorf("failed to get docker network interface: %w", err)
		}

		if systemInterface == "" {
			systemInterface, err = r.GetSystemInterfaceName(networkName)
			if err != nil {
				return fmt.Errorf("failed to get system interface name: %w", err)
			}
		}

		err = r.StartTcpdump(r.ctx, systemInterface)
		if err != nil {
			return fmt.Errorf("failed to start tcpdump: %w", err)
		}
	}

	r.mu.Lock()
	now := time.Now()
	r.startTime = &now
	r.mu.Unlock()

	defer r.StopTcpdump()

	select {
	case <-time.After(time.Duration(r.config.SimulationTime+1) * time.Second):
		log.Println("Simulation completed")
	case <-r.ctx.Done():
		log.Println("Simulation cancelled")
	}

	log.Println("Stopping network traffic capture...")
	return nil
}

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

func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return fmt.Errorf("no scenario is running")
	}

	if r.cancel != nil {
		r.cancel()
	}

	if err := r.StopTcpdump(); err != nil {
		log.Printf("Error stopping tcpdump: %v", err)
	}

	if err := r.StopDockerCompose(); err != nil {
		log.Printf("Error stopping docker compose: %v", err)
	}

	if err := r.CleanConfigFolder(); err != nil {
		log.Printf("Error cleaning config folder: %v", err)
	}

	return nil
}

func Start(dockerComposePath string, simulationTime int, outputFile, configPath string) (string, error) {
	runner := GetGlobalRunner()

	if runner.IsRunning() {
		return "", fmt.Errorf("a scenario is already running")
	}

	runner.Configure(dockerComposePath, simulationTime, outputFile, configPath)

	go func() {
		if err := runner.Run(); err != nil {
			log.Printf("Error running scenario: %v", err)
		}
	}()

	absPath, err := filepath.Abs(runner.outputFile)
	if err != nil {
		return runner.outputFile, nil // Return relative path if absolute fails
	}

	return absPath, nil
}

func Stop() error {
	runner := GetGlobalRunner()
	return runner.Stop()
}

func GetStatus() Status {
	runner := GetGlobalRunner()
	return runner.Status()
}

func ExampleUsage() error {
	outputPath, err := Start("docker-compose.yml", 10, "output.pcap", "")
	if err != nil {
		return err
	}

	log.Printf("Scenario started, output will be saved to: %s", outputPath)
	return nil
}

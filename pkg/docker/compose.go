package docker

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// Service represents a Docker Compose service configuration
type Service struct {
	Build       *BuildConfig           `yaml:"build,omitempty"`
	Image       string                 `yaml:"image,omitempty"`
	Container   string                 `yaml:"container_name,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	Networks    map[string]NetworkSpec `yaml:"networks,omitempty"`
	Environment []string               `yaml:"environment,omitempty"`
	Expose      []string               `yaml:"expose,omitempty"`
	HealthCheck *HealthCheck           `yaml:"healthcheck,omitempty"`
	DependsOn   map[string]Dependency  `yaml:"depends_on,omitempty"`
}

// BuildConfig represents the build configuration for a service
type BuildConfig struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// NetworkSpec represents network specification for a service
type NetworkSpec struct {
	IPv4Address string `yaml:"ipv4_address,omitempty"`
	MacAddress  string `yaml:"mac_address,omitempty"`
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period"`
}

// Dependency represents service dependency configuration
type Dependency struct {
	Condition string `yaml:"condition"`
}

// Network represents Docker network configuration
type Network struct {
	Name string     `yaml:"name"`
	IPAM IPAMConfig `yaml:"ipam"`
}

// IPAMConfig represents IPAM configuration
type IPAMConfig struct {
	Config []SubnetConfig `yaml:"config"`
}

// SubnetConfig represents subnet configuration
type SubnetConfig struct {
	Subnet string `yaml:"subnet"`
}

// DockerCompose represents the complete Docker Compose file structure
type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks"`
}

// ScenarioNode represents a node in the scenario
type ScenarioNode struct {
	Role string `json:"role" yaml:"role"`
	IP   string `json:"ip,omitempty" yaml:"ip,omitempty"`
	Mac  string `json:"mac,omitempty" yaml:"mac,omitempty"`
}

// Scenario represents the complete scenario configuration
type Scenario struct {
	Protocol  string         `json:"protocol" yaml:"protocol"`
	IPNetwork string         `json:"ip_network" yaml:"ip_network"`
	Nodes     []ScenarioNode `json:"nodes" yaml:"nodes"`
}

// Generator manages Docker Compose file generation and validation
type Generator struct {
	services   map[string]Service
	networks   map[string]Network
	protocol   string
	ipBase     net.IP
	path       string
	configPath string
	lastIP     net.IP
}

// NewGenerator creates a new Docker Compose generator
func NewGenerator(protocol, filePath, configPath string) *Generator {
	return &Generator{
		services:   make(map[string]Service),
		networks:   make(map[string]Network),
		protocol:   protocol,
		path:       filePath,
		configPath: configPath,
	}
}

// AddNetwork adds a network to the Docker Compose configuration
func (g *Generator) AddNetwork(name, ipRange string) error {
	_, ipNet, err := net.ParseCIDR(ipRange)
	if err != nil {
		return fmt.Errorf("invalid IP range %s: %w", ipRange, err)
	}

	g.networks[name] = Network{
		Name: name,
		IPAM: IPAMConfig{
			Config: []SubnetConfig{
				{Subnet: ipRange},
			},
		},
	}

	g.ipBase = ipNet.IP
	g.lastIP = incrementIP(g.ipBase)
	return nil
}

// AddNode adds a node (service) to the Docker Compose configuration
func (g *Generator) AddNode(role string, index int, ip, mac string, dependencies map[string][]int) error {
	var nodeIP net.IP

	if ip == "" {
		g.lastIP = incrementIP(g.lastIP)
		nodeIP = g.lastIP
	} else {
		nodeIP = net.ParseIP(ip)
		if nodeIP == nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}

	// Get the first (and should be only) network name
	var networkName string
	for name := range g.networks {
		networkName = name
		break
	}

	if networkName == "" {
		return fmt.Errorf("no network configured")
	}

	// Build service configuration
	service := Service{
		Build: &BuildConfig{
			Context:    fmt.Sprintf("./protocols/%s/%s", g.protocol, role),
			Dockerfile: fmt.Sprintf("Dockerfile.%s", role),
		},
		Image:     fmt.Sprintf("%s_%s_image", g.protocol, role),
		Container: fmt.Sprintf("%s_%s_container_%d", g.protocol, role, index),
		Volumes: []string{
			fmt.Sprintf("%s/%ss/%d/%s.%s:/app/%s.%s:ro",
				g.configPath, role, index, role,
				getConfigExtension(role), role, getConfigExtension(role)),
		},
		Networks: map[string]NetworkSpec{
			networkName: {
				IPv4Address: nodeIP.String(),
			},
		},
		Environment: []string{"PYTHONUNBUFFERED=1"},
	}

	// Add role-specific configurations
	if role == "slave" {
		service.Expose = []string{"502"}
		service.HealthCheck = &HealthCheck{
			Test:        []string{"CMD-SHELL", "test -f /app/app_running.lock"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "10s",
		}
	}

	// Add dependencies
	if dependencies != nil {
		service.DependsOn = make(map[string]Dependency)
		for depRole, indices := range dependencies {
			for _, depIndex := range indices {
				depServiceName := fmt.Sprintf("%s_%s_%d", g.protocol, depRole, depIndex)
				service.DependsOn[depServiceName] = Dependency{
					Condition: "service_healthy",
				}
			}
		}
	}

	// Add MAC address if provided
	if mac != "" {
		service.Networks[networkName] = NetworkSpec{
			IPv4Address: nodeIP.String(),
			MacAddress:  mac,
		}
	}

	serviceName := fmt.Sprintf("%s_%s_%d", g.protocol, role, index)
	g.services[serviceName] = service

	return nil
}

// Generate creates the Docker Compose YAML file
func (g *Generator) Generate() error {
	compose := DockerCompose{
		Services: g.services,
		Networks: g.networks,
	}

	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = os.WriteFile(g.path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Validate validates the generated Docker Compose file
func (g *Generator) Validate() bool {
	exists := fileExists(g.path)
	if !exists {
		if err := g.Generate(); err != nil {
			return false
		}
	}

	result := validateFile(g.path)

	if !exists {
		os.Remove(g.path)
	}

	return result
}

// ValidateFile validates a given Docker Compose file
func ValidateFile(filePath string) bool {
	return validateFile(filePath)
}

// Parse generates a Docker Compose configuration based on the provided scenario
func (g *Generator) Parse(scenario Scenario, dockerComposePath, scenarioConfigPath string) error {
	g.path = dockerComposePath
	g.configPath = scenarioConfigPath

	// Add network
	err := g.AddNetwork("icscommemulator", scenario.IPNetwork)
	if err != nil {
		return fmt.Errorf("failed to add network: %w", err)
	}

	// Get dependencies for master nodes
	masterDependencies := getDependencies(scenario.Nodes)

	// Add master nodes
	masterIndex := 0
	for _, node := range scenario.Nodes {
		if isMaster(node) {
			err := g.AddNode(node.Role, masterIndex, node.IP, node.Mac, masterDependencies)
			if err != nil {
				return fmt.Errorf("failed to add master node: %w", err)
			}
			masterIndex++
		}
	}

	// Add slave nodes
	slaveIndex := 0
	for _, node := range scenario.Nodes {
		if isSlave(node) {
			err := g.AddNode(node.Role, slaveIndex, node.IP, node.Mac, nil)
			if err != nil {
				return fmt.Errorf("failed to add slave node: %w", err)
			}
			slaveIndex++
		}
	}

	// Generate the file
	err = g.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate docker-compose file: %w", err)
	}

	// Validate the generated file
	if !g.Validate() {
		return fmt.Errorf("invalid docker-compose file generated")
	}

	return nil
}

// Helper functions

func getConfigExtension(role string) string {
	if role == "slave" {
		return "yaml"
	}
	return "csv"
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func validateFile(filePath string) bool {
	cmd := exec.Command("docker", "compose", "-f", filePath, "config")
	err := cmd.Run()
	return err == nil
}

func getDependencies(nodes []ScenarioNode) map[string][]int {
	dependencies := make(map[string][]int)
	var slaves []int

	slaveIndex := 0
	for _, node := range nodes {
		if isSlave(node) {
			slaves = append(slaves, slaveIndex)
			slaveIndex++
		}
	}

	if len(slaves) > 0 {
		dependencies["slave"] = slaves
	}

	return dependencies
}

func isMaster(node ScenarioNode) bool {
	return node.Role == "master"
}

func isSlave(node ScenarioNode) bool {
	return node.Role == "slave"
}

// ParseScenario is a convenience function to parse a scenario from raw data
func ParseScenario(scenarioData []byte, protocol, dockerComposePath, scenarioConfigPath string) error {
	var scenario Scenario
	err := json.Unmarshal(scenarioData, &scenario)
	if err != nil {
		return fmt.Errorf("failed to unmarshal scenario: %w", err)
	}

	generator := NewGenerator(protocol, dockerComposePath, scenarioConfigPath)
	return generator.Parse(scenario, dockerComposePath, scenarioConfigPath)
}

// Example usage function (equivalent to the Python __main__ block)
func ExampleUsage() error {
	scenario := Scenario{
		Protocol:  "modbus",
		IPNetwork: "172.28.0.0/16",
		Nodes: []ScenarioNode{
			{Role: "master", IP: "172.28.0.2"},
			{Role: "slave", IP: "172.28.0.3"},
			{Role: "slave", IP: "172.28.0.4"},
		},
	}

	generator := NewGenerator("modbus", "docker-compose.yml", "/tmp/ICSCommEmulator")
	return generator.Parse(scenario, "docker-compose.yml", "/tmp/ICSCommEmulator")
}

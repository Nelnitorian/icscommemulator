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
	Build       *BuildConfig          `yaml:"build,omitempty"`
	Image       string                `yaml:"image,omitempty"`
	Container   string                `yaml:"container_name,omitempty"`
	Volumes     []string              `yaml:"volumes,omitempty"`
	Networks    map[string]NetworkSpec `yaml:"networks,omitempty"`
	Environment []string              `yaml:"environment,omitempty"`
	Expose      []string              `yaml:"expose,omitempty"`
	HealthCheck *HealthCheck          `yaml:"healthcheck,omitempty"`
	DependsOn   map[string]Dependency `yaml:"depends_on,omitempty"`
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

// HealthCheck represents the health check configuration for a service
type HealthCheck struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

// Dependency represents a service dependency configuration
type Dependency struct {
	Condition string `yaml:"condition"`
}

// NetworkConfig represents a Docker network configuration
type NetworkConfig struct {
	Driver string              `yaml:"driver"`
	IPAM   IPAMConfig          `yaml:"ipam"`
}

// IPAMConfig represents IP Address Management configuration
type IPAMConfig struct {
	Config []IPAMSubnetConfig `yaml:"config"`
}

// IPAMSubnetConfig represents subnet configuration for IPAM
type IPAMSubnetConfig struct {
	Subnet string `yaml:"subnet"`
}

// DockerCompose represents the complete Docker Compose configuration
type DockerCompose struct {
	Version  string                   `yaml:"version"`
	Services map[string]Service       `yaml:"services"`
	Networks map[string]NetworkConfig `yaml:"networks"`
}

// Generator manages Docker Compose file generation
type Generator struct {
	compose    *DockerCompose
	configPath string
	protocol   string
}

// NewGenerator creates a new Docker Compose generator
func NewGenerator(configPath, protocol string) *Generator {
	return &Generator{
		compose: &DockerCompose{
			Version:  "3.8",
			Services: make(map[string]Service),
			Networks: make(map[string]NetworkConfig),
		},
		configPath: configPath,
		protocol:   protocol,
	}
}

// AddNetwork adds a network configuration to the Docker Compose
func (g *Generator) AddNetwork(name, subnet string) {
	g.compose.Networks[name] = NetworkConfig{
		Driver: "bridge",
		IPAM: IPAMConfig{
			Config: []IPAMSubnetConfig{
				{Subnet: subnet},
			},
		},
	}
}

// getProtocolDockerfiles returns the Dockerfile paths for master and slave based on protocol
func (g *Generator) getProtocolDockerfiles() (string, string) {
    switch g.protocol {
    case "modbus":
        return "./protocols/modbus/master/Dockerfile.master", "./protocols/modbus/slave/Dockerfile.slave"
    case "dnp3":
        return "./protocols/dnp3/master/Dockerfile.master", "./protocols/dnp3/slave/Dockerfile.slave"
    case "iec104":
        return "./protocols/iec104/master/Dockerfile.master", "./protocols/iec104/slave/Dockerfile.slave"
    default:
        return "", ""
	}
}

// AddMasterService adds a master service to the Docker Compose
func (g *Generator) AddMasterService(index int, ip, networkName string) error {
    // masterDockerfile, _ := g.getProtocolDockerfiles()
    
    serviceName := fmt.Sprintf("master_%d", index)
    configVolume := fmt.Sprintf("%s/masters/%d:/app/config", g.configPath, index)

    // Context should be the directory containing the Dockerfile
    buildContext := fmt.Sprintf("./protocols/%s/master", g.protocol)
    
    service := Service{
        Build: &BuildConfig{
            Context:    buildContext,
            Dockerfile: "Dockerfile.master",  // Relative to context
        },
        Container: serviceName,
        Volumes:   []string{configVolume},
        Networks: map[string]NetworkSpec{
            networkName: {IPv4Address: ip},
        },
    }

    // Protocol-specific environment variables or configurations
    switch g.protocol {
    case "modbus":
        service.Environment = []string{"PROTOCOL=modbus"}
    case "dnp3":
        service.Environment = []string{"PROTOCOL=dnp3", "LOGLEVEL=INFO"}
    case "iec104":
        service.Environment = []string{"PROTOCOL=iec104", "LOGLEVEL=INFO"}
    }

    g.compose.Services[serviceName] = service
    return nil
}

// AddSlaveService adds a slave service to the Docker Compose
func (g *Generator) AddSlaveService(index int, ip, networkName string, port int, dependencies []string) error {
    // _, slaveDockerfile := g.getProtocolDockerfiles()
    
    serviceName := fmt.Sprintf("slave_%d", index)
    configVolume := fmt.Sprintf("%s/slaves/%d:/app/config", g.configPath, index)

    // Context should be the directory containing the Dockerfile
    buildContext := fmt.Sprintf("./protocols/%s/slave", g.protocol)

    service := Service{
        Build: &BuildConfig{
            Context:    buildContext,
            Dockerfile: "Dockerfile.slave",  // Relative to context
        },
        Container: serviceName,
        Volumes:   []string{configVolume},
        Networks: map[string]NetworkSpec{
            networkName: {IPv4Address: ip},
        },
        Expose: []string{fmt.Sprintf("%d", port)},
    }

    // Protocol-specific configurations
    switch g.protocol {
    case "modbus":
        service.Environment = []string{"PROTOCOL=modbus"}
        service.HealthCheck = &HealthCheck{
            Test:     []string{"CMD-SHELL", fmt.Sprintf("nc -zv localhost %d || exit 1", port)},
            Interval: "10s",
            Timeout:  "5s",
            Retries:  3,
        }
    case "dnp3":
        service.Environment = []string{"PROTOCOL=dnp3", "LOGLEVEL=INFO"}
        service.HealthCheck = &HealthCheck{
            Test:     []string{"CMD-SHELL", fmt.Sprintf("nc -zv localhost %d || exit 1", port)},
            Interval: "10s",
            Timeout:  "5s",
            Retries:  3,
        }
    case "iec104":
        service.Environment = []string{"PROTOCOL=iec104", "LOGLEVEL=INFO"}
        service.HealthCheck = &HealthCheck{
            Test:     []string{"CMD-SHELL", fmt.Sprintf("nc -zv localhost %d || exit 1", port)},
            Interval: "10s",
            Timeout:  "5s",
            Retries:  3,
        }
    }

    // Add dependencies if provided
    if len(dependencies) > 0 {
        service.DependsOn = make(map[string]Dependency)
        for _, dep := range dependencies {
            service.DependsOn[dep] = Dependency{Condition: "service_healthy"}
        }
    }

    g.compose.Services[serviceName] = service
    return nil
}


// Generate creates the Docker Compose YAML file
func (g *Generator) Generate(outputPath string) error {
	data, err := yaml.Marshal(g.compose)
	if err != nil {
		return fmt.Errorf("failed to marshal Docker Compose: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write Docker Compose file: %w", err)
	}

	return nil
}

// ValidateCompose validates the generated Docker Compose file
func ValidateCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", composePath, "config")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose validation failed: %s - %w", string(output), err)
	}
	return nil
}

// GetNetworkSubnet calculates the network subnet from an IP and network size
func GetNetworkSubnet(baseIP string, prefixLen int) (string, error) {
	ip := net.ParseIP(baseIP)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", baseIP)
	}

	// Create the network CIDR
	mask := net.CIDRMask(prefixLen, 32)
	ipNet := &net.IPNet{
		IP:   ip.Mask(mask),
		Mask: mask,
	}

	return ipNet.String(), nil
}

// ParsePort extracts port number from various types
func ParsePort(port interface{}) (int, error) {
	switch v := port.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		var p int
		_, err := fmt.Sscanf(v, "%d", &p)
		if err != nil {
			return 0, fmt.Errorf("failed to parse port: %w", err)
		}
		return p, nil
	default:
		return 0, fmt.Errorf("unsupported port type: %T", port)
	}
}

// ScenarioNode represents a node from the scenario configuration
type ScenarioNode struct {
	Role string      `json:"role"`
	IP   string      `json:"ip"`
	Port interface{} `json:"port"`
}

// GenerateFromScenario generates Docker Compose from a scenario configuration
func GenerateFromScenario(scenarioPath, configPath, outputPath, protocol string) error {
	// Read scenario file
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		return fmt.Errorf("failed to read scenario file: %w", err)
	}

	// Parse scenario
	var scenario struct {
		Protocol  string         `json:"protocol" yaml:"protocol"`
		IPNetwork string         `json:"ip_network" yaml:"ip_network"`
		Nodes     []ScenarioNode `json:"nodes" yaml:"nodes"`
	}

	// Try YAML first, then JSON
	err = yaml.Unmarshal(data, &scenario)
	if err != nil {
		err = json.Unmarshal(data, &scenario)
		if err != nil {
			return fmt.Errorf("failed to parse scenario file: %w", err)
		}
	}

	// Use protocol from scenario if not provided
	if protocol == "" {
		protocol = scenario.Protocol
	}

	// Create generator
	generator := NewGenerator(configPath, protocol)

	// Add network
	networkName := "ics_network"
	generator.AddNetwork(networkName, scenario.IPNetwork)

	// Track slaves for dependencies
	var slaveServices []string

	// Add services
	masterIndex := 0
	slaveIndex := 0

	for _, node := range scenario.Nodes {
		if node.Role == "master" {
			err := generator.AddMasterService(masterIndex, node.IP, networkName)
			if err != nil {
				return fmt.Errorf("failed to add master service: %w", err)
			}
			masterIndex++
		} else if node.Role == "slave" {
			port, err := ParsePort(node.Port)
			if err != nil {
				// Use default port based on protocol
				switch protocol {
				case "modbus":
					port = 502
				case "dnp3":
					port = 20000
				case "iec104":
					port = 2404
				default:
					port = 502
				}
			}

			var deps []string
			if slaveIndex > 0 {
				deps = []string{slaveServices[slaveIndex-1]}
			}

			err = generator.AddSlaveService(slaveIndex, node.IP, networkName, port, deps)
			if err != nil {
				return fmt.Errorf("failed to add slave service: %w", err)
			}

			slaveServices = append(slaveServices, fmt.Sprintf("slave_%d", slaveIndex))
			slaveIndex++
		}
	}

	// Generate the compose file
	return generator.Generate(outputPath)
}

// UpCompose starts the Docker Compose services
func UpCompose(composePath string, detached bool) error {
	args := []string{"compose", "-f", composePath, "up"}
	if detached {
		args = append(args, "-d")
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DownCompose stops and removes the Docker Compose services
func DownCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", composePath, "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetComposeStatus returns the status of Docker Compose services
func GetComposeStatus(composePath string) (string, error) {
	cmd := exec.Command("docker", "compose", "-f", composePath, "ps")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get compose status: %w", err)
	}
	return string(output), nil
}

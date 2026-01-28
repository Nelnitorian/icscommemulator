package service

import (
	"os"
	"path/filepath"
	"testing"

	"icscommemulator/pkg/adapter"
	"icscommemulator/pkg/runner"
)

type fakeRunner struct {
	startCalled bool
	lastConfig  string
	lastNetwork *runner.NetworkEmulation
}

func (f *fakeRunner) Start(_ string, _ int, _ string, configPath string, network *runner.NetworkEmulation) (string, error) {
	f.startCalled = true
	f.lastConfig = configPath
	f.lastNetwork = network
	return filepath.Join(configPath, "outputs", "fake.pcap"), nil
}
func (f *fakeRunner) Stop() error { return nil }
func (f *fakeRunner) GetStatus() runner.Status {
	return runner.Status{}
}
func (f *fakeRunner) IsRunning() bool { return false }

func TestRunScenarioModbus(t *testing.T) {
	workDir := t.TempDir()
	fake := &fakeRunner{}
	svc := NewScenarioService(workDir, fake)

	data := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master_0", Role: "master", Name: "master_0", IP: "192.168.1.10"}},
			{Data: adapter.NodeData{
				ID:               "slave_0",
				Role:             "slave",
				Name:             "slave_0",
				IP:               "192.168.1.11",
				Port:             "502",
				SlaveID:          "1",
				HoldingRegisters: map[string]int{"0": 100},
			}},
		},
		Edges: []adapter.Edge{
			{Data: adapter.EdgeData{
				ID:     "edge_0",
				Source: "master_0",
				Target: "slave_0",
				Messages: []adapter.Message{
					{Timestamp: 0, Recurrent: false, Interval: nil, FunctionCode: 3, StartAddress: 0, Count: 1},
				},
			}},
		},
	}

	if _, err := svc.RunScenario(RunScenarioRequest{CytoscapeData: data, SimulationTime: 1}); err != nil {
		t.Fatalf("RunScenario failed: %v", err)
	}

	if !fake.startCalled {
		t.Fatalf("runner Start not called")
	}

	if fake.lastConfig == "" {
		t.Fatalf("runner config path not set")
	}

	if _, err := os.Stat(filepath.Join(fake.lastConfig, "masters", "0", "master.yaml")); err != nil {
		t.Fatalf("master.yaml not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fake.lastConfig, "slaves", "0", "slave.yaml")); err != nil {
		t.Fatalf("slave.yaml not created: %v", err)
	}
}

func TestRunScenarioIEC104(t *testing.T) {
	workDir := t.TempDir()
	fake := &fakeRunner{}
	svc := NewScenarioService(workDir, fake)

	data := adapter.CytoscapeData{
		Protocol:  "iec104",
		IPNetwork: "10.0.0.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master_0", Role: "master", Name: "master_0", IP: "10.0.0.10"}},
			{Data: adapter.NodeData{
				ID:            "slave_0",
				Role:          "slave",
				Name:          "slave_0",
				IP:            "10.0.0.11",
				Port:          "2404",
				CommonAddress: "1",
				SinglePoints: map[string]adapter.IEC104Point{
					"1001": {IOA: 1001, Value: true, ReportMS: 0},
				},
			}},
		},
		Edges: []adapter.Edge{
			{Data: adapter.EdgeData{
				ID:     "edge_0",
				Source: "master_0",
				Target: "slave_0",
				Messages: []adapter.Message{
					{Timestamp: 0, Recurrent: false, Interval: nil, TypeID: 45, CommonAddress: 1, IOA: 1001, COT: 6, Value: "1"},
				},
			}},
		},
	}

	if _, err := svc.RunScenario(RunScenarioRequest{CytoscapeData: data, SimulationTime: 1}); err != nil {
		t.Fatalf("RunScenario failed: %v", err)
	}

	if !fake.startCalled {
		t.Fatalf("runner Start not called")
	}

	if fake.lastConfig == "" {
		t.Fatalf("runner config path not set")
	}

	if _, err := os.Stat(filepath.Join(fake.lastConfig, "masters", "0", "master.yaml")); err != nil {
		t.Fatalf("master.yaml not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fake.lastConfig, "slaves", "0", "slave.yaml")); err != nil {
		t.Fatalf("slave.yaml not created: %v", err)
	}
}

func TestRunScenarioNetworkEmulation(t *testing.T) {
	workDir := t.TempDir()
	fake := &fakeRunner{}
	svc := NewScenarioService(workDir, fake)

	data := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master_0", Role: "master", Name: "master_0", IP: "192.168.1.10"}},
			{Data: adapter.NodeData{
				ID:               "slave_0",
				Role:             "slave",
				Name:             "slave_0",
				IP:               "192.168.1.11",
				Port:             "502",
				SlaveID:          "1",
				HoldingRegisters: map[string]int{"0": 100},
			}},
		},
		Edges: []adapter.Edge{
			{Data: adapter.EdgeData{
				ID:     "edge_0",
				Source: "master_0",
				Target: "slave_0",
				Messages: []adapter.Message{
					{Timestamp: 0, Recurrent: false, Interval: nil, FunctionCode: 3, StartAddress: 0, Count: 1},
				},
			}},
		},
	}

	network := &NetworkEmulation{RateLimitMBps: 5.5, PacketLossPercent: 1.2}
	if _, err := svc.RunScenario(RunScenarioRequest{
		CytoscapeData:  data,
		SimulationTime: 1,
		Network:        network,
	}); err != nil {
		t.Fatalf("RunScenario failed: %v", err)
	}

	if fake.lastNetwork == nil {
		t.Fatalf("expected network emulation to be forwarded to runner")
	}
	if fake.lastNetwork.RateLimitMBps != network.RateLimitMBps {
		t.Fatalf("expected rate limit %v, got %v", network.RateLimitMBps, fake.lastNetwork.RateLimitMBps)
	}
	if fake.lastNetwork.PacketLossPercent != network.PacketLossPercent {
		t.Fatalf("expected packet loss %v, got %v", network.PacketLossPercent, fake.lastNetwork.PacketLossPercent)
	}
}

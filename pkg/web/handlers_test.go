package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"icscommemulator/pkg/adapter"
	"icscommemulator/pkg/runner"
	"icscommemulator/pkg/service"
)

type fakeStorage struct {
	SaveScenarioFunc              func(name string, data adapter.CytoscapeData) error
	GetCytoscapeScenarioAsMapFunc func(name string) (map[string]interface{}, error)
	GetCreatedScenariosFunc       func() ([]string, error)
	CheckScenarioExistsFunc       func(name string) bool
	DeleteScenarioFunc            func(name string) error
}

func (f *fakeStorage) SaveScenario(name string, data adapter.CytoscapeData) error {
	if f.SaveScenarioFunc != nil {
		return f.SaveScenarioFunc(name, data)
	}
	return nil
}

func (f *fakeStorage) GetCytoscapeScenario(name string) (adapter.CytoscapeData, error) {
	return adapter.CytoscapeData{}, nil
}

func (f *fakeStorage) GetCytoscapeScenarioAsMap(name string) (map[string]interface{}, error) {
	if f.GetCytoscapeScenarioAsMapFunc != nil {
		return f.GetCytoscapeScenarioAsMapFunc(name)
	}
	return nil, nil
}

func (f *fakeStorage) GetCreatedScenarios() ([]string, error) {
	if f.GetCreatedScenariosFunc != nil {
		return f.GetCreatedScenariosFunc()
	}
	return nil, nil
}

func (f *fakeStorage) CheckScenarioExists(name string) bool {
	if f.CheckScenarioExistsFunc != nil {
		return f.CheckScenarioExistsFunc(name)
	}
	return false
}

func (f *fakeStorage) DeleteScenario(name string) error {
	if f.DeleteScenarioFunc != nil {
		return f.DeleteScenarioFunc(name)
	}
	return nil
}

type fakeRunner struct {
	startCalled bool
	stopCalled  bool
	status      runner.Status
	startErr    error
	stopErr     error
	lastConfig  string
}

func (f *fakeRunner) Start(_ string, _ int, _ string, configPath string, _ *runner.NetworkEmulation) (string, error) {
	f.startCalled = true
	f.lastConfig = configPath
	return filepath.Join(configPath, "outputs", "fake.pcap"), f.startErr
}

func (f *fakeRunner) Stop() error {
	f.stopCalled = true
	return f.stopErr
}

func (f *fakeRunner) GetStatus() runner.Status {
	return f.status
}

func (f *fakeRunner) IsRunning() bool {
	return f.startCalled && !f.stopCalled
}

func newTestScenarioService(t *testing.T, fr *fakeRunner) *service.ScenarioService {
	t.Helper()
	return service.NewScenarioService(t.TempDir(), fr)
}

func TestHandleNetworksGetList(t *testing.T) {
	storage := &fakeStorage{
		GetCreatedScenariosFunc: func() ([]string, error) {
			return []string{"alpha", "beta"}, nil
		},
	}

	handlers := NewHandlers(storage, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/networks/", nil)
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got []string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 2 || got[0] != "alpha" {
		t.Fatalf("unexpected response: %v", got)
	}
}

func TestHandleNetworksGetNetwork(t *testing.T) {
	expected := map[string]interface{}{
		"protocol": "modbus",
		"nodes":    []interface{}{},
	}

	storage := &fakeStorage{
		GetCytoscapeScenarioAsMapFunc: func(name string) (map[string]interface{}, error) {
			if name != "demo" {
				t.Fatalf("unexpected name: %s", name)
			}
			return expected, nil
		},
	}

	handlers := NewHandlers(storage, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/networks/demo", nil)
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got["protocol"] != "modbus" {
		t.Fatalf("unexpected response: %v", got)
	}
}

func TestHandleNetworksCreateNetwork(t *testing.T) {
	var savedName string
	var savedData adapter.CytoscapeData

	storage := &fakeStorage{
		CheckScenarioExistsFunc: func(name string) bool { return false },
		SaveScenarioFunc: func(name string, data adapter.CytoscapeData) error {
			savedName = name
			savedData = data
			return nil
		},
	}

	handlers := NewHandlers(storage, nil)
	body, _ := json.Marshal(NetworkRequest{
		ProjectName: "demo",
		IPSubrange:  "192.168.10.0/24",
		Protocol:    "modbus",
		MasterNodes: 1,
		SlaveNodes:  2,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/networks/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if savedName != "demo" {
		t.Fatalf("expected scenario name demo, got %s", savedName)
	}
	if savedData.Protocol != "modbus" || savedData.IPNetwork != "192.168.10.0/24" {
		t.Fatalf("unexpected saved data: %+v", savedData)
	}
	if len(savedData.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(savedData.Nodes))
	}
}

func TestHandleNetworksUpdateNetwork(t *testing.T) {
	var savedName string
	var savedData adapter.CytoscapeData

	storage := &fakeStorage{
		SaveScenarioFunc: func(name string, data adapter.CytoscapeData) error {
			savedName = name
			savedData = data
			return nil
		},
	}

	handlers := NewHandlers(storage, nil)
	data := adapter.CytoscapeData{
		Protocol:  "modbus",
		IPNetwork: "192.168.1.0/24",
		Nodes: []adapter.Node{
			{Data: adapter.NodeData{ID: "master_0", Role: "master", Name: "master_0", IP: "192.168.1.10"}},
			{Data: adapter.NodeData{ID: "slave_0", Role: "slave", Name: "slave_0", IP: "192.168.1.11", SlaveID: "1"}},
		},
		Edges: []adapter.Edge{
			{Data: adapter.EdgeData{ID: "edge_0", Source: "master_0", Target: "slave_0"}},
		},
	}

	body, _ := json.Marshal(data)
	req := httptest.NewRequest(http.MethodPut, "/api/networks/demo", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if savedName != "demo" {
		t.Fatalf("expected saved name demo, got %s", savedName)
	}
	if savedData.Protocol != "modbus" || savedData.IPNetwork != "192.168.1.0/24" {
		t.Fatalf("unexpected saved data: %+v", savedData)
	}
}

func TestHandleNetworksUpdateRequiresName(t *testing.T) {
	handlers := NewHandlers(&fakeStorage{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/networks/", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleNetworksDeleteNetwork(t *testing.T) {
	var deleted string
	storage := &fakeStorage{
		DeleteScenarioFunc: func(name string) error {
			deleted = name
			return nil
		},
	}

	handlers := NewHandlers(storage, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/networks/demo", nil)
	rec := httptest.NewRecorder()

	handlers.HandleNetworks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if deleted != "demo" {
		t.Fatalf("expected deleted demo, got %s", deleted)
	}
}

func TestHandleRunValidationErrors(t *testing.T) {
	handlers := NewHandlers(&fakeStorage{}, newTestScenarioService(t, &fakeRunner{}))
	req := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader([]byte(`{"ip_network":"192.168.1.0/24","simulation_time":1}`)))
	rec := httptest.NewRecorder()

	handlers.HandleRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRunInvalidNetworkConfig(t *testing.T) {
	handlers := NewHandlers(&fakeStorage{}, newTestScenarioService(t, &fakeRunner{}))
	req := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader([]byte(`{
		"protocol":"modbus",
		"ip_network":"192.168.1.0/24",
		"nodes":[],
		"edges":[],
		"simulation_time":1,
		"network":{"packet_loss_percent":150}
	}`)))
	rec := httptest.NewRecorder()

	handlers.HandleRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRunGetStatus(t *testing.T) {
	fr := &fakeRunner{status: runner.Status{ElapsedSeconds: 3, TotalSeconds: 10, Running: true}}
	handlers := NewHandlers(&fakeStorage{}, newTestScenarioService(t, fr))

	req := httptest.NewRequest(http.MethodGet, "/api/run", nil)
	rec := httptest.NewRecorder()

	handlers.HandleRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var got runner.Status
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode status: %v", err)
	}
	if got.ElapsedSeconds != 3 || got.TotalSeconds != 10 || !got.Running {
		t.Fatalf("unexpected status: %+v", got)
	}
}

func TestHandleRunStopScenario(t *testing.T) {
	fr := &fakeRunner{}
	handlers := NewHandlers(&fakeStorage{}, newTestScenarioService(t, fr))

	req := httptest.NewRequest(http.MethodDelete, "/api/run", nil)
	rec := httptest.NewRecorder()

	handlers.HandleRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !fr.stopCalled {
		t.Fatalf("expected StopScenario to be called")
	}
}

func TestHandleRunScenarioSuccess(t *testing.T) {
	fr := &fakeRunner{}
	handlers := NewHandlers(&fakeStorage{}, newTestScenarioService(t, fr))

	request := RunRequest{
		Protocol:       "modbus",
		IPNetwork:      "192.168.1.0/24",
		SimulationTime: 1,
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

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.HandleRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !fr.startCalled {
		t.Fatalf("expected scenario runner to start")
	}
}

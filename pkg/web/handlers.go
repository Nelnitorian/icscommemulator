package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"icscommemulator/pkg/adapter"
	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/scenario"
	"icscommemulator/pkg/service"
)

type Handlers struct {
	scenarioStorage scenario.Storage
	scenarioService *service.ScenarioService
}

func NewHandlers(storage scenario.Storage, scenarioSvc *service.ScenarioService) *Handlers {
	return &Handlers{
		scenarioStorage: storage,
		scenarioService: scenarioSvc,
	}
}

func (h *Handlers) HandleNetworks(w http.ResponseWriter, r *http.Request) {
	path := extractPathSegment(r.URL.Path, "/api/networks/")

	logger.Debug("Received %s request to %s (path: '%s')", r.Method, r.URL.Path, path)

	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.handleGetNetworks(w, r)
		} else {
			h.handleGetNetwork(w, r, path)
		}
	case http.MethodPost:
		h.handleCreateNetwork(w, r)
	case http.MethodPut:
		if path != "" {
			h.handleUpdateNetwork(w, r, path)
		} else {
			sendError(w, "Network name required for PUT", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if path != "" {
			h.handleDeleteNetwork(w, r, path)
		} else {
			sendError(w, "Network name required for DELETE", http.StatusBadRequest)
		}
	default:
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) HandleImportNetwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(25 << 20); err != nil {
		sendError(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendError(w, "Scenario file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}
	name = sanitizeScenarioName(name)
	if name == "" {
		sendError(w, "Scenario name is required", http.StatusBadRequest)
		return
	}

	if h.scenarioStorage.CheckScenarioExists(name) {
		sendError(w, "Project already exists", http.StatusBadRequest)
		return
	}

	tmpFile, err := os.CreateTemp("", "ics-import-*"+filepath.Ext(header.Filename))
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		sendError(w, fmt.Sprintf("Failed to store scenario file: %v", err), http.StatusInternalServerError)
		return
	}

	if err := scenario.ImportScenarioFile(name, tmpFile.Name()); err != nil {
		sendError(w, fmt.Sprintf("Failed to import scenario: %v", err), http.StatusBadRequest)
		return
	}

	sendJSON(w, APIResponse{
		Status:  200,
		Message: fmt.Sprintf("Scenario imported as %s", name),
	})
}

func sanitizeScenarioName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	name = re.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_-")
	return name
}

func (h *Handlers) handleGetNetworks(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.scenarioStorage.GetCreatedScenarios()
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to get scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	sendJSON(w, scenarios)
}

func (h *Handlers) handleGetNetwork(w http.ResponseWriter, r *http.Request, name string) {
	scenarioData, err := h.scenarioStorage.GetCytoscapeScenarioAsMap(name)
	if err != nil {
		sendError(w, "Network not found", http.StatusNotFound)
		return
	}
	sendJSON(w, scenarioData)
}

func (h *Handlers) handleCreateNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Info("Creating new network scenario")

	var req NetworkRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	if req.ProjectName == "" || req.IPSubrange == "" || req.Protocol == "" {
		sendError(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if req.MasterNodes <= 0 || req.SlaveNodes <= 0 {
		sendError(w, "masterNodes and slaveNodes must be positive integers", http.StatusBadRequest)
		return
	}

	if h.scenarioStorage.CheckScenarioExists(req.ProjectName) {
		sendError(w, "Project already exists", http.StatusBadRequest)
		return
	}

	_, ipNet, err := net.ParseCIDR(req.IPSubrange)
	if err != nil {
		sendError(w, "Invalid IP range", http.StatusBadRequest)
		return
	}

	baseIP := incrementIP(incrementIP(ipNet.IP))
	nodes, edges := adapter.GenerateNetworkNodes(
		adapter.Protocol(req.Protocol),
		baseIP,
		req.MasterNodes,
		req.SlaveNodes,
	)

	networkData := adapter.CytoscapeData{
		Protocol:  req.Protocol,
		IPNetwork: ipNet.String(),
		Nodes:     nodes,
		Edges:     edges,
	}

	if err := h.scenarioStorage.SaveScenario(req.ProjectName, networkData); err != nil {
		sendError(w, fmt.Sprintf("Failed to save scenario: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully created scenario: %s", req.ProjectName)
	sendJSON(w, APIResponse{
		Status:  200,
		Message: fmt.Sprintf("Network created and saved as %s", req.ProjectName),
	})
}

func (h *Handlers) handleUpdateNetwork(w http.ResponseWriter, r *http.Request, name string) {
	logger.Info("Updating network scenario: %s", name)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var data adapter.CytoscapeData
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		sendError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	logs := adapter.ValidateCytoscapeScenario(data, adapter.ERROR)
	if len(logs) > 0 {
		sendError(w, strings.Join(logs, "; "), http.StatusBadRequest)
		return
	}

	if err := h.scenarioStorage.SaveScenario(name, data); err != nil {
		sendError(w, fmt.Sprintf("Failed to save scenario: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully updated scenario: %s", name)
	sendJSON(w, APIResponse{Message: fmt.Sprintf("Scenario saved as %s", name)})
}

func (h *Handlers) handleDeleteNetwork(w http.ResponseWriter, r *http.Request, name string) {
	logger.Info("Deleting network scenario: %s", name)

	if err := h.scenarioStorage.DeleteScenario(name); err != nil {
		sendError(w, fmt.Sprintf("Failed to delete scenario: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("Successfully deleted scenario: %s", name)
	sendJSON(w, APIResponse{Message: fmt.Sprintf("Scenario %s deleted", name)})
}

func (h *Handlers) HandleRun(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleRunScenario(w, r)
	case http.MethodGet:
		h.handleGetRunStatus(w, r)
	case http.MethodDelete:
		h.handleStopScenario(w, r)
	default:
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received run scenario request")

	var req RunRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	if req.Protocol == "" || req.IPNetwork == "" {
		sendError(w, "Missing required fields: protocol and ip_network", http.StatusBadRequest)
		return
	}

	if req.SimulationTime <= 0 {
		sendError(w, "simulation_time must be greater than 0", http.StatusBadRequest)
		return
	}

	cytoscapeData := adapter.CytoscapeData{
		Protocol:  req.Protocol,
		IPNetwork: req.IPNetwork,
		Nodes:     req.Nodes,
		Edges:     req.Edges,
	}

	result, err := h.scenarioService.RunScenario(service.RunScenarioRequest{
		CytoscapeData:  cytoscapeData,
		SimulationTime: req.SimulationTime,
	})

	if err != nil {
		sendError(w, fmt.Sprintf("Error running scenario: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":         result.Message,
		"simulation_time": result.SimulationTime,
		"file_path":       result.FilePath,
	}

	sendJSON(w, response)
}

func (h *Handlers) handleGetRunStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scenarioService.GetScenarioStatus()
	sendJSON(w, status)
}

func (h *Handlers) handleStopScenario(w http.ResponseWriter, r *http.Request) {
	if err := h.scenarioService.StopScenario(); err != nil {
		sendError(w, fmt.Sprintf("Error stopping scenario: %v", err), http.StatusInternalServerError)
		return
	}
	sendJSON(w, APIResponse{Message: "Scenario stopped"})
}

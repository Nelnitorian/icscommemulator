package web

// import (
// 	"encoding/json"
// 	"fmt"
// 	"html/template"
// 	"io"
// 	"log"
// 	"net"
// 	"net/http"
// 	"os"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"gopkg.in/yaml.v3" // <--- IMPORT AÑADIDO

// 	"icscommemulator/pkg/adapter"
// 	"icscommemulator/pkg/config"
// 	"icscommemulator/pkg/docker"
// 	"icscommemulator/pkg/logger"
// 	"icscommemulator/pkg/runner"
// 	"icscommemulator/pkg/scenario"
// )

// // NetworkAPI handles HTTP requests for the ICS Communication Emulator
// type NetworkAPI struct {
// 	mux        *http.ServeMux
// 	templates  *template.Template
// 	staticPath string
// }

// // APIResponse represents a standard API response
// type APIResponse struct {
// 	Status  int                    `json:"status,omitempty"`
// 	Message string                 `json:"message,omitempty"`
// 	Error   string                 `json:"error,omitempty"`
// 	Data    map[string]interface{} `json:"data,omitempty"`
// }

// // NetworkRequest represents a network creation request
// type NetworkRequest struct {
// 	ProjectName string `json:"projectName"`
// 	IPSubrange  string `json:"ipSubrange"`
// 	Protocol    string `json:"protocol"`
// 	MasterNodes int    `json:"masterNodes"`
// 	SlaveNodes  int    `json:"slaveNodes"`
// }

// // RunRequest represents a scenario run request
// type RunRequest struct {
//     Protocol       string          `json:"protocol"`
//     IPNetwork      string          `json:"ip_network"`
//     Nodes          []adapter.Node  `json:"nodes"`
//     Edges          []adapter.Edge  `json:"edges"`
//     SimulationTime int             `json:"simulation_time"`
// }

// // NewNetworkAPI creates a new NetworkAPI instance
// func NewNetworkAPI() *NetworkAPI {
// 	logger.Info("Creating new NetworkAPI instance")
// 	api := &NetworkAPI{
// 		mux:        http.NewServeMux(),
// 		staticPath: "web/static",
// 	}

// 	// Load templates
// 	api.loadTemplates()

// 	// Setup routes
// 	api.setupRoutes()

// 	logger.Info("NetworkAPI instance created successfully")
// 	return api
// }

// // loadTemplates loads HTML templates with custom functions
// func (api *NetworkAPI) loadTemplates() {
// 	logger.Debug("Loading HTML templates")

// 	// Define custom functions for templates
// 	funcMap := template.FuncMap{
// 		"toJSON": func(v interface{}) template.JS {
// 			b, err := json.Marshal(v)
// 			if err != nil {
// 				logger.Error("Failed to marshal data to JSON in template: %v", err)
// 				return template.JS("{}")
// 			}
// 			return template.JS(b)
// 		},
// 	}

// 	templates, err := template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html")
// 	if err != nil {
// 		logger.Warning("Failed to load templates: %v", err)
// 		// Create empty template to avoid nil pointer
// 		api.templates = template.New("empty").Funcs(funcMap)
// 	} else {
// 		api.templates = templates
// 		logger.Debug("Templates loaded successfully")
// 	}
// }

// // setupRoutes configures all HTTP routes
// func (api *NetworkAPI) setupRoutes() {
// 	logger.Debug("Setting up HTTP routes")

// 	// Static files
// 	api.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(api.staticPath))))

// 	// API routes
// 	api.mux.HandleFunc("/api/networks/", api.handleNetworks)
// 	api.mux.HandleFunc("/api/run", api.handleRun)

// 	// Web pages
// 	api.mux.HandleFunc("/", api.handleHome)
// 	api.mux.HandleFunc("/index.html", api.handleHome)
// 	api.mux.HandleFunc("/networks/", api.handleNetworkPage)

// 	logger.Debug("HTTP routes configured")
// }

// // handleNetworks handles /api/networks/ endpoints
// func (api *NetworkAPI) handleNetworks(w http.ResponseWriter, r *http.Request) {
// 	// Log request details
// 	logger.Debug("Received %s request to %s", r.Method, r.URL.Path)
// 	logger.Debug("Request headers: %+v", r.Header)

// 	// Extract network name from URL path
// 	path := strings.TrimPrefix(r.URL.Path, "/api/networks/")
// 	path = strings.Trim(path, "/")
// 	logger.Debug("Extracted path: '%s'", path)

// 	switch r.Method {
// 	case "GET":
// 		if path == "" {
// 			api.handleGetNetworks(w, r)
// 		} else {
// 			api.handleGetNetwork(w, r, path)
// 		}
// 	case "POST":
// 		api.handleCreateNetwork(w, r)
// 	case "PUT":
// 		if path != "" {
// 			api.handleUpdateNetwork(w, r, path)
// 		} else {
// 			logger.Warning("PUT request without network name")
// 			api.sendError(w, "Network name required for PUT", http.StatusBadRequest)
// 		}
// 	default:
// 		logger.Warning("Unsupported method: %s", r.Method)
// 		api.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
// 	}
// }

// // handleCreateNetwork creates a new network scenario
// func (api *NetworkAPI) handleCreateNetwork(w http.ResponseWriter, r *http.Request) {
// 	logger.Info("Creating new network scenario")

// 	// Read body content for debugging
// 	bodyBytes, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		logger.Error("Failed to read request body: %v", err)
// 		api.sendError(w, "Failed to read request body", http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Raw request body: %s", string(bodyBytes))
// 	logger.Debug("Content-Type: %s", r.Header.Get("Content-Type"))
// 	logger.Debug("Content-Length: %s", r.Header.Get("Content-Length"))

// 	// Check if body is empty
// 	if len(bodyBytes) == 0 {
// 		logger.Warning("Request body is empty")
// 		api.sendError(w, "Request body is empty", http.StatusBadRequest)
// 		return
// 	}

// 	// Parse JSON from body
// 	var req NetworkRequest
// 	err = json.Unmarshal(bodyBytes, &req)
// 	if err != nil {
// 		logger.Error("Failed to parse JSON: %v", err)
// 		logger.Debug("Invalid JSON content: %s", string(bodyBytes))
// 		api.sendError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
// 		return
// 	}

// 	logger.DebugStruct("Parsed request", req)

// 	// Validate required fields
// 	if req.ProjectName == "" || req.IPSubrange == "" || req.Protocol == "" {
// 		logger.Warning("Missing required parameters in request")
// 		logger.Debug("ProjectName: '%s', IPSubrange: '%s', Protocol: '%s'",
// 			req.ProjectName, req.IPSubrange, req.Protocol)
// 		api.sendError(w, "Missing parameters", http.StatusBadRequest)
// 		return
// 	}

// 	// Validate node counts
// 	if req.MasterNodes <= 0 || req.SlaveNodes <= 0 {
// 		logger.Warning("Invalid node counts - Masters: %d, Slaves: %d", req.MasterNodes, req.SlaveNodes)
// 		api.sendError(w, "masterNodes and slaveNodes must be positive integers", http.StatusBadRequest)
// 		return
// 	}

// 	// Check if project already exists
// 	if scenario.CheckScenarioExists(req.ProjectName) {
// 		logger.Warning("Project already exists: %s", req.ProjectName)
// 		api.sendError(w, "Project already exists", http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Parsed node counts - Masters: %d, Slaves: %d", req.MasterNodes, req.SlaveNodes)

// 	// Parse IP network
// 	_, ipNet, err := net.ParseCIDR(req.IPSubrange)
// 	if err != nil {
// 		logger.Error("Invalid IP range: %s - %v", req.IPSubrange, err)
// 		api.sendError(w, "Invalid IP range", http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Parsed IP network: %s", ipNet.String())

// 	// Generate network nodes
// 	baseIP := incrementIP(ipNet.IP) // Skip network address
// 	baseIP = incrementIP(baseIP)    // Skip +1 (equivalent to +2 from network address)
// 	logger.Debug("Base IP for node generation: %s", baseIP.String())

// 	nodes, edges := adapter.GenerateNetworkNodes(
// 		adapter.Protocol(req.Protocol),
// 		baseIP,
// 		req.MasterNodes,
// 		req.SlaveNodes,
// 	)

// 	logger.Debug("Generated %d nodes and %d edges", len(nodes), len(edges))

// 	// Create network structure
// 	networkData := adapter.CytoscapeData{
// 		Protocol:  req.Protocol,
// 		IPNetwork: ipNet.String(),
// 		Nodes:     nodes,
// 		Edges:     edges,
// 	}

// 	// Save scenario
// 	err = scenario.SaveScenario(req.ProjectName, networkData)
// 	if err != nil {
// 		logger.Error("Failed to save scenario '%s': %v", req.ProjectName, err)
// 		api.sendError(w, fmt.Sprintf("Failed to save scenario: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	logger.Info("Successfully created and saved scenario: %s", req.ProjectName)

// 	response := APIResponse{
// 		Status:  200,
// 		Message: fmt.Sprintf("Network created and saved as %s", req.ProjectName),
// 	}

// 	api.sendJSON(w, response)
// }

// // handleGetNetworks returns all scenarios
// func (api *NetworkAPI) handleGetNetworks(w http.ResponseWriter, _ *http.Request) {
// 	scenarios, err := scenario.GetCreatedScenarios()
// 	if err != nil {
// 		api.sendError(w, fmt.Sprintf("Failed to get scenarios: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	api.sendJSON(w, scenarios)
// }

// // handleGetNetwork returns a specific scenario
// func (api *NetworkAPI) handleGetNetwork(w http.ResponseWriter, r *http.Request, name string) {
// 	scenarioData, err := scenario.GetCytoscapeScenarioAsMap(name)
// 	if err != nil {
// 		api.sendError(w, "Network not found", http.StatusNotFound)
// 		return
// 	}

// 	api.sendJSON(w, scenarioData)
// }

// // handleUpdateNetwork updates an existing network scenario
// func (api *NetworkAPI) handleUpdateNetwork(w http.ResponseWriter, r *http.Request, name string) {
// 	logger.Info("Updating network scenario: %s", name)

// 	// Read body content for debugging
// 	bodyBytes, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		logger.Error("Error reading request body: %v", err)
// 		api.sendError(w, "Error reading request body", http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Raw PUT body length: %d bytes", len(bodyBytes))
// 	logger.Debug("Raw PUT body: %s", string(bodyBytes))

// 	// Step 1: Parse to generic map to validate JSON structure
// 	var tempMap map[string]interface{}
// 	err = json.Unmarshal(bodyBytes, &tempMap)
// 	if err != nil {
// 		logger.Error("Error parsing JSON to map: %v", err)
// 		api.sendError(w, fmt.Sprintf("Invalid JSON structure: %v", err), http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Successfully parsed JSON to map")
// 	logger.DebugStruct("JSON structure", tempMap)

// 	// Step 2: Try to parse to CytoscapeData struct with detailed error reporting
// 	var data adapter.CytoscapeData
// 	err = json.Unmarshal(bodyBytes, &data)
// 	if err != nil {
// 		logger.Error("Error unmarshaling to CytoscapeData: %v", err)

// 		// Additional debugging for specific error types
// 		if syntaxErr, ok := err.(*json.SyntaxError); ok {
// 			logger.Error("JSON syntax error at byte offset %d", syntaxErr.Offset)
// 		}
// 		if unmarshalErr, ok := err.(*json.UnmarshalTypeError); ok {
// 			logger.Error("JSON unmarshal type error: cannot unmarshal %s into Go struct field %s of type %s at offset %d",
// 				unmarshalErr.Value, unmarshalErr.Field, unmarshalErr.Type, unmarshalErr.Offset)
// 		}

// 		api.sendError(w, fmt.Sprintf("Error validating JSON structure: %v", err), http.StatusBadRequest)
// 		return
// 	}

// 	logger.Debug("Successfully parsed JSON to CytoscapeData")
// 	logger.DebugStruct("Parsed CytoscapeData", data)

// 	// Step 3: Validate scenario
// 	logs := adapter.ValidateCytoscapeScenario(data, adapter.ERROR)
// 	if len(logs) > 0 {
// 		logger.Warning("Scenario validation failed: %v", logs)
// 		api.sendError(w, strings.Join(logs, "; "), http.StatusBadRequest)
// 		return
// 	}

// 	// Step 4: Save scenario
// 	err = scenario.SaveScenario(name, data)
// 	if err != nil {
// 		logger.Error("Failed to save scenario '%s': %v", name, err)
// 		api.sendError(w, fmt.Sprintf("Failed to save scenario: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	logger.Info("Successfully updated scenario: %s", name)

// 	response := APIResponse{
// 		Message: fmt.Sprintf("Scenario saved as %s", name),
// 	}

// 	api.sendJSON(w, response)
// }

// // handleRun handles /api/run/ endpoints
// func (api *NetworkAPI) handleRun(w http.ResponseWriter, r *http.Request) {
//     switch r.Method {
//     case "POST":
//         api.handleRunScenario(w, r)
//     case "GET":
//         api.handleGetRunStatus(w, r)
//     case "DELETE":
//         api.handleStopScenario(w, r)
//     default:
//         api.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
//     }
// }

// // handleRunScenario starts a scenario execution
// func (api *NetworkAPI) handleRunScenario(w http.ResponseWriter, r *http.Request) {
//     var req RunRequest
//     err := json.NewDecoder(r.Body).Decode(&req)
//     if err != nil {
//         api.sendError(w, "Invalid JSON", http.StatusBadRequest)
//         return
//     }

//     // Validate required fields
//     if req.Protocol == "" || req.IPNetwork == "" {
//         api.sendError(w, "Missing required fields: protocol and ip_network", http.StatusBadRequest)
//         return
//     }

//     if req.SimulationTime <= 0 {
//         api.sendError(w, "simulation_time must be greater than 0", http.StatusBadRequest)
//         return
//     }

//     // Create CytoscapeData from the request
//     cytoscapeData := adapter.CytoscapeData{
//         Protocol:  req.Protocol,
//         IPNetwork: req.IPNetwork,
//         Nodes:     req.Nodes,
//         Edges:     req.Edges,
//     }

//     // Validate the scenario data
//     logs := adapter.ValidateCytoscapeScenario(cytoscapeData, adapter.ERROR)
//     if len(logs) > 0 {
//         logger.Warning("Scenario validation failed: %v", logs)
//         api.sendError(w, strings.Join(logs, "; "), http.StatusBadRequest)
//         return
//     }

//     // Convert CytoscapeData to map for docker generation
//     dataBytes, err := json.Marshal(cytoscapeData)
//     if err != nil {
//         api.sendError(w, fmt.Sprintf("Error converting scenario data: %v", err), http.StatusInternalServerError)
//         return
//     }
//     var scenarioData map[string]interface{}
//     json.Unmarshal(dataBytes, &scenarioData)

//     dockerComposePath := "docker-compose.yml"
//     configPath := "/tmp/ICSCommEmulator"

//     // Generate Docker Compose
//     err = api.generateDockerCompose(scenarioData, dockerComposePath, configPath)
//     if err != nil {
//         api.sendError(w, fmt.Sprintf("Error generating docker-compose: %v", err), http.StatusInternalServerError)
//         return
//     }

//     // Generate scenario configuration
//     err = api.generateScenarioConfig(scenarioData, configPath)
//     if err != nil {
//         api.sendError(w, fmt.Sprintf("Error generating scenario config: %v", err), http.StatusInternalServerError)
//         return
//     }

//     // Generate PCAP filename with timestamp to avoid collisions
//     pcapFilename := fmt.Sprintf("scenario_%d.pcap", time.Now().Unix())

//     // Start scenario
//     filePath, err := runner.Start(dockerComposePath, req.SimulationTime, pcapFilename, configPath)
//     if err != nil {
//         api.sendError(w, fmt.Sprintf("Error running scenario: %v", err), http.StatusInternalServerError)
//         return
//     }

//     response := map[string]interface{}{
//         "message":         "Scenario running",
//         "simulation_time": req.SimulationTime,
//         "file_path":       filePath,
//     }

//     api.sendJSON(w, response)
// }

// // handleGetRunStatus returns the status of running scenario
// func (api *NetworkAPI) handleGetRunStatus(w http.ResponseWriter, r *http.Request) {
// 	status := runner.GetStatus()
// 	api.sendJSON(w, status)
// }

// // handleStopScenario stops the running scenario
// func (api *NetworkAPI) handleStopScenario(w http.ResponseWriter, r *http.Request) {
// 	err := runner.Stop()
// 	if err != nil {
// 		api.sendError(w, fmt.Sprintf("Error stopping scenario: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	response := APIResponse{
// 		Message: "Scenario stopped",
// 	}

// 	api.sendJSON(w, response)
// }

// // handleHome serves the home page
// func (api *NetworkAPI) handleHome(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
// 		http.NotFound(w, r)
// 		return
// 	}

// 	err := api.templates.ExecuteTemplate(w, "index.html", nil)
// 	if err != nil {
// 		http.Error(w, "Template error", http.StatusInternalServerError)
// 		log.Printf("Template execution error: %v", err)
// 	}
// }

// // handleNetworkPage serves the network page
// func (api *NetworkAPI) handleNetworkPage(w http.ResponseWriter, r *http.Request) {
// 	// Extract network ID from URL
// 	path := strings.TrimPrefix(r.URL.Path, "/networks/")
// 	path = strings.Trim(path, "/")

// 	if path == "" {
// 		http.Error(w, "Network ID required", http.StatusBadRequest)
// 		return
// 	}

// 	// Get network data
// 	networkData, err := scenario.GetCytoscapeScenarioAsMap(path)
// 	if err != nil {
// 		http.NotFound(w, r)
// 		return
// 	}

// 	data := struct {
// 		NetworkData map[string]interface{}
// 	}{
// 		NetworkData: networkData,
// 	}

// 	err = api.templates.ExecuteTemplate(w, "network.html", data)
// 	if err != nil {
// 		http.Error(w, "Template error", http.StatusInternalServerError)
// 		log.Printf("Template execution error: %v", err)
// 	}
// }

// // Helper methods

// // sendJSON sends a JSON response
// func (api *NetworkAPI) sendJSON(w http.ResponseWriter, data interface{}) {
// 	logger.Debug("Sending JSON response")
// 	w.Header().Set("Content-Type", "application/json")
// 	if err := json.NewEncoder(w).Encode(data); err != nil {
// 		logger.Error("Failed to encode JSON response: %v", err)
// 	}
// }

// // sendError sends an error response
// func (api *NetworkAPI) sendError(w http.ResponseWriter, message string, status int) {
// 	logger.Warning("Sending error response: %d - %s", status, message)
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(status)

// 	response := APIResponse{
// 		Status: status,
// 		Error:  message,
// 	}

// 	if err := json.NewEncoder(w).Encode(response); err != nil {
// 		logger.Error("Failed to encode error response: %v", err)
// 	}
// }

// // generateDockerCompose generates docker-compose configuration
// func (api *NetworkAPI) generateDockerCompose(scenarioData map[string]interface{}, dockerComposePath, scenarioConfigPath string) error {
//     protocol, ok := scenarioData["protocol"].(string)
//     if !ok {
//         return fmt.Errorf("invalid protocol in scenario")
//     }

//     // Extract nodes from Cytoscape format
//     nodesInterface, ok := scenarioData["nodes"].([]interface{})
//     if !ok {
//         return fmt.Errorf("invalid nodes format in scenario")
//     }

//     // Convert Cytoscape nodes to simple format
//     simpleNodes := []map[string]interface{}{}
//     for _, nodeInterface := range nodesInterface {
//         nodeMap, ok := nodeInterface.(map[string]interface{})
//         if !ok {
//             continue
//         }

//         // Extract data field
//         dataInterface, ok := nodeMap["data"]
//         if ok {
//             if dataMap, ok := dataInterface.(map[string]interface{}); ok {
//                 simpleNodes = append(simpleNodes, dataMap)
//             }
//         }
//     }

//     // Create simplified scenario with extracted data
//     simplifiedScenario := map[string]interface{}{
//         "protocol":   protocol,
//         "ip_network": scenarioData["ip_network"],
//         "nodes":      simpleNodes,
//     }

//     // Create temporary scenario YAML file
//     scenarioPath := filepath.Join(scenarioConfigPath, "scenario.yaml")
//     yamlData, err := yaml.Marshal(simplifiedScenario)
//     if err != nil {
//         return fmt.Errorf("failed to marshal scenario to YAML: %w", err)
//     }

//     // Ensure directory exists
//     if err := os.MkdirAll(scenarioConfigPath, 0755); err != nil {
//         return fmt.Errorf("failed to create scenario config directory: %w", err)
//     }

//     // Write scenario YAML file
//     err = os.WriteFile(scenarioPath, yamlData, 0644)
//     if err != nil {
//         return fmt.Errorf("failed to write scenario file: %w", err)
//     }

//     logger.Debug("Created temporary scenario file: %s", scenarioPath)

//     return docker.GenerateFromScenario(scenarioPath, scenarioConfigPath, dockerComposePath, protocol)
// }

// func (api *NetworkAPI) generateScenarioConfig(scenarioData map[string]interface{}, scenarioConfigPath string) error {
//     logger.Info("Generating scenario configuration in: %s", scenarioConfigPath)
//     logger.Debug("Scenario data for config generation: %+v", scenarioData)

//     // Extract nodes from Cytoscape format
//     nodesInterface, ok := scenarioData["nodes"].([]interface{})
//     if !ok {
//         return fmt.Errorf("invalid nodes format in scenario")
//     }

//     // Convert Cytoscape nodes to simple format (extract data field)
//     simpleNodes := []interface{}{}
//     for _, nodeInterface := range nodesInterface {
//         nodeMap, ok := nodeInterface.(map[string]interface{})
//         if !ok {
//             continue
//         }

//         // Extract data field
//         if dataInterface, ok := nodeMap["data"]; ok {
//             simpleNodes = append(simpleNodes, dataInterface)
//         }
//     }

//     // Extract edges from Cytoscape format
//     edgesInterface, ok := scenarioData["edges"].([]interface{})
//     if !ok {
//         edgesInterface = []interface{}{}
//     }

//     // Process edges to extract messages and map them to nodes
//     masterMessages := make(map[string][]interface{}) // map[masterID]messages

//     for _, edgeInterface := range edgesInterface {
//         edgeMap, ok := edgeInterface.(map[string]interface{})
//         if !ok {
//             continue
//         }

//         dataInterface, ok := edgeMap["data"]
//         if !ok {
//             continue
//         }

//         edgeData, ok := dataInterface.(map[string]interface{})
//         if !ok {
//             continue
//         }

//         // Get source (master), target (slave), and messages
//         sourceID, _ := edgeData["source"].(string)
//         targetID, _ := edgeData["target"].(string)
//         messages, _ := edgeData["messages"].([]interface{})

//         // Find target slave to get IP and port
//         var targetIP string
//         var targetPort interface{}
//         for _, node := range simpleNodes {
//             nodeData, ok := node.(map[string]interface{})
//             if !ok {
//                 continue
//             }
//             if nodeData["id"] == targetID {
//                 targetIP, _ = nodeData["ip"].(string)
//                 targetPort = nodeData["port"]
//                 break
//             }
//         }

//         // Add IP and port to each message
//         for _, msg := range messages {
//             msgMap, ok := msg.(map[string]interface{})
//             if !ok {
//                 continue
//             }

//             // Set target IP and port if not already set
//             if msgMap["ip"] == nil || msgMap["ip"] == "" {
//                 msgMap["ip"] = targetIP
//             }
//             if msgMap["port"] == nil || msgMap["port"] == 0 {
//                 if targetPort != nil {
//                     msgMap["port"] = targetPort
//                 }
//             }

//             // Add this message to the master's messages
//             masterMessages[sourceID] = append(masterMessages[sourceID], msgMap)
//         }
//     }

//     // Merge messages back into nodes
//     for _, node := range simpleNodes {
//         nodeData, ok := node.(map[string]interface{})
//         if !ok {
//             continue
//         }

//         nodeID, _ := nodeData["id"].(string)
//         if messages, ok := masterMessages[nodeID]; ok {
//             nodeData["messages"] = messages
//         }
//     }

//     // Create simplified scenario
//     simplifiedScenario := map[string]interface{}{
//         "protocol": scenarioData["protocol"],
//         "nodes":    simpleNodes,
//     }

//     logger.Debug("Simplified scenario for config generation: %+v", simplifiedScenario)

//     err := config.GenerateFromMap(simplifiedScenario, scenarioConfigPath)
//     if err != nil {
//         logger.Error("Failed to generate scenario config: %v", err)
//         return err
//     }

//     // Verify the generated files exist
//     logger.Debug("Verifying generated configuration files...")

//     // Check if master and slave directories exist
//     mastersDir := filepath.Join(scenarioConfigPath, "masters")
//     slavesDir := filepath.Join(scenarioConfigPath, "slaves")

//     if _, err := os.Stat(mastersDir); err != nil {
//         logger.Warning("Masters directory not found: %s", mastersDir)
//     } else {
//         logger.Debug("Masters directory exists: %s", mastersDir)
//         if entries, err := os.ReadDir(mastersDir); err == nil {
//             for _, entry := range entries {
//                 logger.Debug("Masters dir entry: %s (isDir: %v)", entry.Name(), entry.IsDir())
//             }
//         }
//     }

//     if _, err := os.Stat(slavesDir); err != nil {
//         logger.Warning("Slaves directory not found: %s", slavesDir)
//     } else {
//         logger.Debug("Slaves directory exists: %s", slavesDir)
//         if entries, err := os.ReadDir(slavesDir); err == nil {
//             for _, entry := range entries {
//                 logger.Debug("Slaves dir entry: %s (isDir: %v)", entry.Name(), entry.IsDir())
//             }
//         }
//     }

//     logger.Info("Scenario configuration generation completed")
//     return nil
// }

// // incrementIP increments an IP address by 1
// func incrementIP(ip net.IP) net.IP {
// 	result := make(net.IP, len(ip))
// 	copy(result, ip)
// 	for i := len(result) - 1; i >= 0; i-- {
// 		result[i]++
// 		if result[i] > 0 {
// 			break
// 		}
// 	}
// 	return result
// }

// // Run starts the HTTP server
// func (api *NetworkAPI) Run(host string, port int) error {
// 	addr := fmt.Sprintf("%s:%d", host, port)
// 	log.Printf("Starting server on %s", addr)
// 	return http.ListenAndServe(addr, api.mux)
// }

// // ServeHTTP implements http.Handler interface
// func (api *NetworkAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	api.mux.ServeHTTP(w, r)
// }

package web

import "icscommemulator/pkg/adapter"

// APIResponse represents a standard API response
type APIResponse struct {
	Status  int                    `json:"status,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// NetworkRequest represents a network creation request
type NetworkRequest struct {
	ProjectName string `json:"projectName" validate:"required"`
	IPSubrange  string `json:"ipSubrange" validate:"required,cidr"`
	Protocol    string `json:"protocol" validate:"required,oneof=modbus dnp3 iec104"`
	MasterNodes int    `json:"masterNodes" validate:"required,min=1"`
	SlaveNodes  int    `json:"slaveNodes" validate:"required,min=1"`
}

// RunRequest represents a scenario run request
type RunRequest struct {
	Protocol       string         `json:"protocol" validate:"required"`
	IPNetwork      string         `json:"ip_network" validate:"required"`
	Nodes          []adapter.Node `json:"nodes" validate:"required,dive"`
	Edges          []adapter.Edge `json:"edges"`
	SimulationTime int            `json:"simulation_time" validate:"required,min=1"`
	Network        *NetworkConfig `json:"network,omitempty"`
}

// NetworkConfig defines optional network emulation controls for a run.
type NetworkConfig struct {
	RateLimitMBps     float64 `json:"rate_limit_mbytes_per_sec,omitempty"`
	PacketLossPercent float64 `json:"packet_loss_percent,omitempty"`
}

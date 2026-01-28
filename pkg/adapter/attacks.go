package adapter

type AttackConfig struct {
	ID          string                 `json:"id" yaml:"id"`
	TechniqueID string                 `json:"technique_id,omitempty" yaml:"technique_id,omitempty"`
	Name        string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool                   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	TargetID    string                 `json:"target_id,omitempty" yaml:"target_id,omitempty"`
	StartTime   int                    `json:"start_time,omitempty" yaml:"start_time,omitempty"`
	Interval    int                    `json:"interval,omitempty" yaml:"interval,omitempty"`
	Count       int                    `json:"count,omitempty" yaml:"count,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

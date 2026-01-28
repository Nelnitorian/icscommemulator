package appconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/scenario"
)

// Config defines the application configuration structure.
type Config struct {
	Scenarios []ScenarioImport `json:"scenarios" yaml:"scenarios"`
}

// ScenarioImport defines a scenario import entry.
type ScenarioImport struct {
	Name      string `json:"name" yaml:"name"`
	Path      string `json:"path" yaml:"path"`
	Overwrite bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
}

// Load reads and parses a config file (YAML or JSON).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file as YAML or JSON: %w", err)
		}
	}

	return &cfg, nil
}

// ImportScenarios imports scenarios from the config.
func (c *Config) ImportScenarios(storage scenario.Storage, baseDir string) error {
	if len(c.Scenarios) == 0 {
		return nil
	}

	var failures []string
	for _, entry := range c.Scenarios {
		name := strings.TrimSpace(entry.Name)
		if entry.Path == "" {
			failures = append(failures, "scenario import missing path")
			continue
		}

		scenarioPath := entry.Path
		if !filepath.IsAbs(scenarioPath) {
			scenarioPath = filepath.Join(baseDir, scenarioPath)
		}

		if name == "" {
			name = strings.TrimSuffix(filepath.Base(scenarioPath), filepath.Ext(scenarioPath))
		}

		if name == "" {
			failures = append(failures, fmt.Sprintf("scenario import path %s has empty name", scenarioPath))
			continue
		}

		if storage.CheckScenarioExists(name) {
			if !entry.Overwrite {
				logger.Warning("Scenario %s already exists; skipping import", name)
				continue
			}
			if err := storage.DeleteScenario(name); err != nil {
				failures = append(failures, fmt.Sprintf("failed to overwrite scenario %s: %v", name, err))
				continue
			}
		}

		if err := scenario.ImportScenarioFile(name, scenarioPath); err != nil {
			failures = append(failures, fmt.Sprintf("failed to import %s: %v", name, err))
			continue
		}

		logger.Info("Imported scenario %s from %s", name, scenarioPath)
	}

	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}

	return nil
}

package scenario

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ImportScenarioFile reads a scenario file (JSON or YAML) and stores it.
func ImportScenarioFile(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read scenario file: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse scenario file as YAML or JSON: %w", err)
		}
	}

	normalized, err := normalizeScenarioData(raw)
	if err != nil {
		return err
	}

	return SaveScenarioFromMap(name, normalized)
}

func normalizeScenarioData(raw map[string]interface{}) (map[string]interface{}, error) {
	if raw == nil {
		return nil, fmt.Errorf("scenario data is empty")
	}

	if _, ok := raw["protocol"]; !ok {
		return nil, fmt.Errorf("scenario is missing protocol")
	}

	if _, ok := raw["ip_network"]; !ok {
		return nil, fmt.Errorf("scenario is missing ip_network")
	}

	nodesValue, ok := raw["nodes"]
	if !ok {
		return nil, fmt.Errorf("scenario is missing nodes")
	}

	nodesList, ok := nodesValue.([]interface{})
	if !ok {
		return nil, fmt.Errorf("scenario nodes are in an invalid format")
	}

	if len(nodesList) == 0 {
		raw["edges"] = ensureEdges(raw)
		return raw, nil
	}

	firstNode, _ := nodesList[0].(map[string]interface{})
	if firstNode != nil {
		if _, ok := firstNode["data"]; ok {
			raw["edges"] = ensureEdges(raw)
			return raw, nil
		}

		if _, ok := firstNode["role"]; ok {
			return convertFlatScenario(raw, nodesList), nil
		}
	}

	return nil, fmt.Errorf("unsupported scenario node format")
}

func convertFlatScenario(raw map[string]interface{}, nodesList []interface{}) map[string]interface{} {
	convertedNodes := make([]interface{}, 0, len(nodesList))
	for _, node := range nodesList {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}
		classes := ""
		if role, ok := nodeMap["role"].(string); ok {
			classes = role
		}
		convertedNodes = append(convertedNodes, map[string]interface{}{
			"data":    nodeMap,
			"classes": classes,
		})
	}

	return map[string]interface{}{
		"protocol":   raw["protocol"],
		"ip_network": raw["ip_network"],
		"nodes":      convertedNodes,
		"edges":      []interface{}{},
	}
}

func ensureEdges(raw map[string]interface{}) interface{} {
	if edges, ok := raw["edges"]; ok {
		return edges
	}
	return []interface{}{}
}

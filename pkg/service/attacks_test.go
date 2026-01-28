package service

import (
	"os"
	"path/filepath"
	"testing"

	"icscommemulator/pkg/adapter"
)

func TestApplyAttacksCoversCatalog(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	attackPath := filepath.Join(wd, "..", "..", "web", "static", "attacks", "attacks.json")
	t.Setenv("ICS_ATTACKS_CATALOG_PATH", attackPath)

	catalog, err := loadAttackCatalog()
	if err != nil {
		t.Fatalf("failed to load attack catalog: %v", err)
	}

	for _, def := range catalog.Attacks {
		def := def
		t.Run(def.Protocol+"_"+def.ID, func(t *testing.T) {
			simplified := buildSimplifiedScenarioForAttack(def)

			svc := &ScenarioService{}
			labels, err := svc.applyAttacks(simplified, def.Protocol)
			if err != nil {
				t.Fatalf("applyAttacks failed: %v", err)
			}

			master := findNodeByRole(simplified, "master")
			if master == nil {
				t.Fatalf("master node not found")
			}

			messages := extractMessages(master)
			expectedCount := 1
			if def.ScheduleDefaults.Count > 0 {
				expectedCount = def.ScheduleDefaults.Count
			}

			if len(messages) != expectedCount {
				t.Fatalf("expected %d messages, got %d", expectedCount, len(messages))
			}
			if len(labels) != 1 {
				t.Fatalf("expected 1 label, got %d", len(labels))
			}
		})
	}
}

func TestApplyAttacksValuesCount(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	attackPath := filepath.Join(wd, "..", "..", "web", "static", "attacks", "attacks.json")
	t.Setenv("ICS_ATTACKS_CATALOG_PATH", attackPath)

	catalog, err := loadAttackCatalog()
	if err != nil {
		t.Fatalf("failed to load attack catalog: %v", err)
	}

	cases := map[string]int{
		"modbus_write_multiple_registers": 3,
		"modbus_write_multiple_coils":     3,
	}

	for _, def := range catalog.Attacks {
		expectedCount, ok := cases[def.ID]
		if !ok {
			continue
		}
		def := def
		t.Run(def.ID, func(t *testing.T) {
			simplified := buildSimplifiedScenarioForAttack(def)
			svc := &ScenarioService{}
			if _, err := svc.applyAttacks(simplified, def.Protocol); err != nil {
				t.Fatalf("applyAttacks failed: %v", err)
			}

			master := findNodeByRole(simplified, "master")
			messages := extractMessages(master)
			if len(messages) == 0 {
				t.Fatalf("expected at least one message")
			}

			countRaw, ok := messages[0]["count"]
			if !ok {
				t.Fatalf("expected count in message")
			}
			count := getInt(countRaw, 0)
			if count != expectedCount {
				t.Fatalf("expected count %d, got %d", expectedCount, count)
			}
		})
	}
}

func buildSimplifiedScenarioForAttack(def AttackDefinition) map[string]interface{} {
	slave := map[string]interface{}{
		"id":             "slave_0",
		"role":           "slave",
		"ip":             "192.168.1.11",
		"port":           defaultPortFor(def.Protocol),
		"slave_id":       1,
		"master_id":      1,
		"outstation_id":  1,
		"common_address": 1,
	}

	master := map[string]interface{}{
		"id":        "master_0",
		"role":      "master",
		"ip":        "192.168.1.10",
		"port":      defaultPortFor(def.Protocol),
		"master_id": 1,
		"messages":  []map[string]interface{}{},
		"attacks": []adapter.AttackConfig{
			{
				ID:      def.ID,
				Enabled: true,
			},
		},
	}

	return map[string]interface{}{
		"nodes": []map[string]interface{}{master, slave},
	}
}

func findNodeByRole(simplified map[string]interface{}, role string) map[string]interface{} {
	nodes, ok := simplified["nodes"].([]map[string]interface{})
	if !ok {
		return nil
	}
	for _, node := range nodes {
		if getString(node["role"]) == role {
			return node
		}
	}
	return nil
}

func extractMessages(node map[string]interface{}) []map[string]interface{} {
	if node == nil {
		return nil
	}
	if messages, ok := node["messages"].([]map[string]interface{}); ok {
		return messages
	}
	raw, ok := node["messages"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if msg, ok := item.(map[string]interface{}); ok {
			out = append(out, msg)
		}
	}
	return out
}

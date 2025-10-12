package scenario

import (
	"icscommemulator/pkg/adapter"
)

// Storage define la interfaz para almacenamiento de escenarios
type Storage interface {
	SaveScenario(name string, data adapter.CytoscapeData) error
	GetCytoscapeScenario(name string) (adapter.CytoscapeData, error)
	GetCytoscapeScenarioAsMap(name string) (map[string]interface{}, error)
	GetCreatedScenarios() ([]string, error)
	CheckScenarioExists(name string) bool
	DeleteScenario(name string) error
}

// storage implementa Storage usando el filesystem
type storage struct {
	rootFolder string
}

// NewStorage crea una nueva instancia de storage
func NewStorage() Storage {
	return &storage{
		rootFolder: SCENARIO_ROOT_FOLDER,
	}
}

// Implementación usando las funciones existentes
func (s *storage) SaveScenario(name string, data adapter.CytoscapeData) error {
	return SaveScenario(name, data)
}

func (s *storage) GetCytoscapeScenario(name string) (adapter.CytoscapeData, error) {
	return GetCytoscapeScenario(name)
}

func (s *storage) GetCytoscapeScenarioAsMap(name string) (map[string]interface{}, error) {
	return GetCytoscapeScenarioAsMap(name)
}

func (s *storage) GetCreatedScenarios() ([]string, error) {
	return GetCreatedScenarios()
}

func (s *storage) CheckScenarioExists(name string) bool {
	return CheckScenarioExists(name)
}

func (s *storage) DeleteScenario(name string) error {
	return DeleteScenario(name)
}

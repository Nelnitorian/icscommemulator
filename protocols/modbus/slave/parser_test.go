package main

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func newTestSlave() *ModbusSlave {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	return &ModbusSlave{
		logger: logger,
	}
}

func TestParseSequentialValuesSupportsTypes(t *testing.T) {
	slave := newTestSlave()

	values := slave.parseSequentialValues([]interface{}{1, float64(2), "3"})
	if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
		t.Fatalf("unexpected sequential values: %v", values)
	}

	fromString := slave.parseSequentialValues("4, 5,6")
	if len(fromString) != 3 || fromString[0] != 4 || fromString[1] != 5 || fromString[2] != 6 {
		t.Fatalf("unexpected sequential values from string: %v", fromString)
	}
}

func TestParseSparseValuesSupportsTypes(t *testing.T) {
	slave := newTestSlave()

	values := slave.parseSparseValues(map[string]interface{}{
		"0": 1,
		"1": float64(2),
		"2": "3",
	})

	if values[0] != 1 || values[1] != 2 || values[2] != 3 {
		t.Fatalf("unexpected sparse values: %v", values)
	}
}

func TestParseSparseValuesWithInterfaceKeys(t *testing.T) {
	slave := newTestSlave()

	values := slave.parseSparseValues(map[interface{}]interface{}{
		0:   7,
		"1": "8",
	})

	if values[0] != 7 || values[1] != 8 {
		t.Fatalf("unexpected sparse values: %v", values)
	}
}

func TestParseRegisterConfigUsesSparseOrSequential(t *testing.T) {
	slave := newTestSlave()

	sequential, sparse := slave.parseRegisterConfig(RegisterConfig{
		Type:   "sequential",
		Values: []interface{}{1, 2},
	})
	if len(sequential) != 2 || len(sparse) != 0 {
		t.Fatalf("unexpected sequential parse: seq=%v sparse=%v", sequential, sparse)
	}

	sequential, sparse = slave.parseRegisterConfig(RegisterConfig{
		Type: "sparse",
		Values: map[string]interface{}{
			"10": 1,
		},
	})
	if len(sequential) != 0 || len(sparse) != 1 || sparse[10] != 1 {
		t.Fatalf("unexpected sparse parse: seq=%v sparse=%v", sequential, sparse)
	}
}

func TestGetPortSupportsTypes(t *testing.T) {
	slave := newTestSlave()

	slave.config.Port = 1502
	if port := slave.getPort(); port != 1502 {
		t.Fatalf("expected port 1502, got %d", port)
	}

	slave.config.Port = "1602"
	if port := slave.getPort(); port != 1602 {
		t.Fatalf("expected port 1602, got %d", port)
	}

	slave.config.Port = float64(1702)
	if port := slave.getPort(); port != 1702 {
		t.Fatalf("expected port 1702, got %d", port)
	}
}

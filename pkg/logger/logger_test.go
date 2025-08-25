// pkg/logger/logger_test.go - CORREGIDO COMPLETAMENTE
package logger

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARNING", WARNING},
		{"warn", WARNING},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"invalid", INFO}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevel(tt.input)
			assert.Equal(t, tt.expected, currentLevel)
		})
	}
}

// Helper function to capture logger output
func captureLoggerOutput(f func()) string {
	// Save original outputs
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Create pipe to capture output
	r, w, _ := os.Pipe()

	// Replace stdout and stderr
	os.Stdout = w
	os.Stderr = w

	// Also redirect the logger outputs to the pipe
	debugLogger.SetOutput(w)
	infoLogger.SetOutput(w)
	warnLogger.SetOutput(w)
	errorLogger.SetOutput(w)

	// Create a channel to collect output
	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	// Execute function
	f()

	// Close writer and restore outputs
	w.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	// Restore logger outputs
	debugLogger.SetOutput(os.Stdout)
	infoLogger.SetOutput(os.Stdout)
	warnLogger.SetOutput(os.Stdout)
	errorLogger.SetOutput(os.Stderr)

	// Return captured output
	return <-outChan
}

func TestLoggingLevels(t *testing.T) {
	// Test DEBUG level
	SetLevel("DEBUG")
	
	output := captureLoggerOutput(func() {
		Debug("debug message")
		Info("info message")
		Warning("warning message")
		Error("error message")
	})
	
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "error message")

	// Test ERROR level (should only show errors)
	SetLevel("ERROR")
	
	output = captureLoggerOutput(func() {
		Debug("debug message")
		Info("info message")
		Warning("warning message")
		Error("error message")
	})
	
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")  
	assert.NotContains(t, output, "warning message")
	assert.Contains(t, output, "error message")
}

func TestDebugStruct(t *testing.T) {
	SetLevel("DEBUG")

	testStruct := struct {
		Name string
		Age  int
	}{"Test", 25}

	output := captureLoggerOutput(func() {
		DebugStruct("TestObject", testStruct)
	})
	
	assert.Contains(t, output, "TestObject")
	assert.Contains(t, output, "Test")
	assert.Contains(t, output, "25")
}

package logger

import "testing"

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
		{"invalid", INFO},
	}

	original := defaultLogger.currentLevel
	t.Cleanup(func() {
		defaultLogger.currentLevel = original
	})

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevel(tt.input)
			if defaultLogger.currentLevel != tt.expected {
				t.Fatalf("expected level %v, got %v", tt.expected, defaultLogger.currentLevel)
			}
		})
	}
}

func TestFormatMessageWithFields(t *testing.T) {
	l := &logger{
		currentLevel: INFO,
		fields: map[string]interface{}{
			"scenario": "demo",
			"node":     3,
		},
	}

	msg := l.formatMessage("hello")
	if msg == "hello" {
		t.Fatalf("expected fields to be appended to message")
	}
	if msg != "hello scenario=demo node=3" && msg != "hello node=3 scenario=demo" {
		t.Fatalf("unexpected formatted message: %s", msg)
	}
}

func TestWithFieldsCopiesState(t *testing.T) {
	base := &logger{
		currentLevel: INFO,
		fields: map[string]interface{}{
			"service": "ics",
		},
	}

	child := base.WithFields(map[string]interface{}{"node": 1}).(*logger)
	if child == base {
		t.Fatalf("expected new logger instance")
	}
	if child.fields["service"] != "ics" || child.fields["node"] != 1 {
		t.Fatalf("unexpected fields: %+v", child.fields)
	}
	if _, ok := base.fields["node"]; ok {
		t.Fatalf("base logger should not be mutated")
	}
}

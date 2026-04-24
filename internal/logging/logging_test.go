package logging

import "testing"

func TestNew_ReturnsLogger(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO", "unknown", ""}

	for _, lvl := range levels {
		t.Run("level="+lvl, func(t *testing.T) {
			logger := New(lvl)
			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
		})
	}
}

func TestLogger_ImplementsInterface(t *testing.T) {
	// Compile-time check: slogLogger must implement Logger.
	var _ Logger = New("info")
}

func TestLogger_Methods_DoNotPanic(t *testing.T) {
	logger := New("debug")

	tests := []struct {
		name string
		fn   func()
	}{
		{"Info", func() { logger.Info("info message", "key", "value") }},
		{"Error", func() { logger.Error("error message", "key", "value") }},
		{"Debug", func() { logger.Debug("debug message", "key", "value") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("method panicked: %v", r)
				}
			}()
			tc.fn()
		})
	}
}

package platform

import (
	"log/slog"
	"testing"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"default on empty", ""},
		{"default on invalid", "verbose"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitLogger(tt.level)

			if Logger == nil {
				t.Fatal("Logger is nil after InitLogger")
			}

			Logger.Info("test log from InitLogger")
		})
	}
}

func TestInitLoggerCanLogAtAllLevels(t *testing.T) {
	InitLogger("debug")

	if Logger == nil {
		t.Fatal("Logger is nil")
	}

	Logger.Debug("debug test")
	Logger.Info("info test")
	Logger.Warn("warn test")
	Logger.Error("error test")
}

func TestGlobalLoggerDefault(t *testing.T) {
	orig := Logger

	InitLogger("info")

	if Logger == nil {
		t.Fatal("Logger should not be nil after InitLogger")
	}
	if slog.Default() == nil {
		t.Fatal("slog default should not be nil after InitLogger")
	}

	Logger = orig
}

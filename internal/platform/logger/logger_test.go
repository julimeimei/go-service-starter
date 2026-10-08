package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewReturnsTextLoggerOutsideProduction(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	log, err := New(Options{
		Level:  "info",
		Env:    "development",
		Output: &output,
	})
	if err != nil {
		t.Fatalf("expected logger to be created, got error: %v", err)
	}

	log.Info("service started", "component", "test")

	line := output.String()
	if !strings.Contains(line, "msg=\"service started\"") {
		t.Fatalf("expected text log to include message, got %q", line)
	}

	if !strings.Contains(line, "component=test") {
		t.Fatalf("expected text log to include attribute, got %q", line)
	}
}

func TestNewReturnsJSONLoggerInProduction(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	log, err := New(Options{
		Level:  "info",
		Env:    "production",
		Output: &output,
	})
	if err != nil {
		t.Fatalf("expected logger to be created, got error: %v", err)
	}

	log.Info("service started", "component", "test")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log, got error: %v", err)
	}

	if entry["msg"] != "service started" {
		t.Fatalf("expected JSON log message service started, got %v", entry["msg"])
	}

	if entry["component"] != "test" {
		t.Fatalf("expected JSON log component test, got %v", entry["component"])
	}
}

func TestNewRespectsLogLevel(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	log, err := New(Options{
		Level:  "warn",
		Env:    "development",
		Output: &output,
	})
	if err != nil {
		t.Fatalf("expected logger to be created, got error: %v", err)
	}

	log.Info("hidden")
	log.Warn("visible")

	line := output.String()
	if strings.Contains(line, "hidden") {
		t.Fatalf("expected info message to be filtered, got %q", line)
	}

	if !strings.Contains(line, "visible") {
		t.Fatalf("expected warn message to be logged, got %q", line)
	}
}

func TestNewRejectsUnsupportedLogLevel(t *testing.T) {
	t.Parallel()

	if _, err := New(Options{Level: "trace"}); err == nil {
		t.Fatal("expected unsupported log level error, got nil")
	}
}

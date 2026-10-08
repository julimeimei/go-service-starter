package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := load(mapLookup(nil))
	if err != nil {
		t.Fatalf("expected defaults to load successfully, got error: %v", err)
	}

	if cfg.App.Name != "go-service-starter" {
		t.Fatalf("expected default app name, got %q", cfg.App.Name)
	}

	if cfg.App.Env != "development" {
		t.Fatalf("expected default app env development, got %q", cfg.App.Env)
	}

	if cfg.HTTP.Host != "127.0.0.1" {
		t.Fatalf("expected default HTTP host 127.0.0.1, got %q", cfg.HTTP.Host)
	}

	if cfg.HTTP.Port != 8080 {
		t.Fatalf("expected default HTTP port 8080, got %d", cfg.HTTP.Port)
	}

	if cfg.HTTP.ReadTimeout != 10*time.Second {
		t.Fatalf("expected default HTTP read timeout 10s, got %s", cfg.HTTP.ReadTimeout)
	}

	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("expected default HTTP read header timeout 5s, got %s", cfg.HTTP.ReadHeaderTimeout)
	}

	if cfg.HTTP.WriteTimeout != 10*time.Second {
		t.Fatalf("expected default HTTP write timeout 10s, got %s", cfg.HTTP.WriteTimeout)
	}

	if cfg.HTTP.IdleTimeout != 60*time.Second {
		t.Fatalf("expected default HTTP idle timeout 60s, got %s", cfg.HTTP.IdleTimeout)
	}

	if cfg.HTTP.MaxBodyBytes != 1048576 {
		t.Fatalf("expected default max body bytes 1048576, got %d", cfg.HTTP.MaxBodyBytes)
	}

	if cfg.HTTP.Addr() != "127.0.0.1:8080" {
		t.Fatalf("expected default HTTP addr 127.0.0.1:8080, got %q", cfg.HTTP.Addr())
	}

	if cfg.Log.Level != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.Log.Level)
	}

	if cfg.Database.URL != "" {
		t.Fatalf("expected default database URL to be empty, got %q", cfg.Database.URL)
	}

	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Fatalf("expected default database connect timeout 5s, got %s", cfg.Database.ConnectTimeout)
	}

	if cfg.Database.ReadinessTimeout != 2*time.Second {
		t.Fatalf("expected default database readiness timeout 2s, got %s", cfg.Database.ReadinessTimeout)
	}

	if cfg.Tracing.Enabled {
		t.Fatal("expected tracing to be disabled by default")
	}

	if cfg.Tracing.Exporter != "stdout" {
		t.Fatalf("expected default tracing exporter stdout, got %q", cfg.Tracing.Exporter)
	}

	if cfg.Tracing.SampleRatio != 1 {
		t.Fatalf("expected default tracing sample ratio 1, got %f", cfg.Tracing.SampleRatio)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("expected default shutdown timeout 10s, got %s", cfg.ShutdownTimeout)
	}
}

func TestLoadOverridesFromEnvironment(t *testing.T) {
	t.Parallel()

	cfg, err := load(mapLookup(map[string]string{
		"APP_NAME":                   "orders-api",
		"APP_ENV":                    "staging",
		"HTTP_HOST":                  "0.0.0.0",
		"HTTP_PORT":                  "9090",
		"HTTP_READ_TIMEOUT":          "11s",
		"HTTP_READ_HEADER_TIMEOUT":   "6s",
		"HTTP_WRITE_TIMEOUT":         "12s",
		"HTTP_IDLE_TIMEOUT":          "70s",
		"HTTP_MAX_BODY_BYTES":        "2097152",
		"LOG_LEVEL":                  "debug",
		"DATABASE_URL":               "postgres://localhost:5432/app?sslmode=disable",
		"DATABASE_CONNECT_TIMEOUT":   "3s",
		"DATABASE_READINESS_TIMEOUT": "1s",
		"TRACING_ENABLED":            "true",
		"TRACING_EXPORTER":           "stdout",
		"TRACING_SAMPLE_RATIO":       "0.5",
		"SHUTDOWN_TIMEOUT":           "15s",
	}))
	if err != nil {
		t.Fatalf("expected overrides to load successfully, got error: %v", err)
	}

	if cfg.App.Name != "orders-api" {
		t.Fatalf("expected overridden app name orders-api, got %q", cfg.App.Name)
	}

	if cfg.App.Env != "staging" {
		t.Fatalf("expected overridden app env staging, got %q", cfg.App.Env)
	}

	if cfg.HTTP.Addr() != "0.0.0.0:9090" {
		t.Fatalf("expected overridden HTTP addr 0.0.0.0:9090, got %q", cfg.HTTP.Addr())
	}

	if cfg.HTTP.ReadTimeout != 11*time.Second {
		t.Fatalf("expected overridden HTTP read timeout 11s, got %s", cfg.HTTP.ReadTimeout)
	}

	if cfg.HTTP.ReadHeaderTimeout != 6*time.Second {
		t.Fatalf("expected overridden HTTP read header timeout 6s, got %s", cfg.HTTP.ReadHeaderTimeout)
	}

	if cfg.HTTP.WriteTimeout != 12*time.Second {
		t.Fatalf("expected overridden HTTP write timeout 12s, got %s", cfg.HTTP.WriteTimeout)
	}

	if cfg.HTTP.IdleTimeout != 70*time.Second {
		t.Fatalf("expected overridden HTTP idle timeout 70s, got %s", cfg.HTTP.IdleTimeout)
	}

	if cfg.HTTP.MaxBodyBytes != 2097152 {
		t.Fatalf("expected overridden max body bytes 2097152, got %d", cfg.HTTP.MaxBodyBytes)
	}

	if cfg.Log.Level != "debug" {
		t.Fatalf("expected overridden log level debug, got %q", cfg.Log.Level)
	}

	if cfg.Database.URL == "" {
		t.Fatal("expected overridden database URL to be set")
	}

	if cfg.Database.ConnectTimeout != 3*time.Second {
		t.Fatalf("expected overridden database connect timeout 3s, got %s", cfg.Database.ConnectTimeout)
	}

	if cfg.Database.ReadinessTimeout != time.Second {
		t.Fatalf("expected overridden database readiness timeout 1s, got %s", cfg.Database.ReadinessTimeout)
	}

	if !cfg.Tracing.Enabled {
		t.Fatal("expected tracing to be enabled")
	}

	if cfg.Tracing.Exporter != "stdout" {
		t.Fatalf("expected tracing exporter stdout, got %q", cfg.Tracing.Exporter)
	}

	if cfg.Tracing.SampleRatio != 0.5 {
		t.Fatalf("expected tracing sample ratio 0.5, got %f", cfg.Tracing.SampleRatio)
	}

	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("expected overridden shutdown timeout 15s, got %s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		env       map[string]string
		wantError string
	}{
		{
			name:      "empty app name",
			env:       map[string]string{"APP_NAME": ""},
			wantError: "APP_NAME is required",
		},
		{
			name:      "invalid app env",
			env:       map[string]string{"APP_ENV": "qa"},
			wantError: "APP_ENV must be one of",
		},
		{
			name:      "empty HTTP host",
			env:       map[string]string{"HTTP_HOST": ""},
			wantError: "HTTP_HOST is required",
		},
		{
			name:      "non integer HTTP port",
			env:       map[string]string{"HTTP_PORT": "abc"},
			wantError: "HTTP_PORT must be an integer",
		},
		{
			name:      "HTTP port below range",
			env:       map[string]string{"HTTP_PORT": "0"},
			wantError: "HTTP_PORT must be between 1 and 65535",
		},
		{
			name:      "HTTP port above range",
			env:       map[string]string{"HTTP_PORT": "65536"},
			wantError: "HTTP_PORT must be between 1 and 65535",
		},
		{
			name:      "invalid HTTP read timeout",
			env:       map[string]string{"HTTP_READ_TIMEOUT": "slow"},
			wantError: "HTTP_READ_TIMEOUT must be a valid duration",
		},
		{
			name:      "zero HTTP read timeout",
			env:       map[string]string{"HTTP_READ_TIMEOUT": "0s"},
			wantError: "HTTP_READ_TIMEOUT must be greater than zero",
		},
		{
			name:      "zero HTTP read header timeout",
			env:       map[string]string{"HTTP_READ_HEADER_TIMEOUT": "0s"},
			wantError: "HTTP_READ_HEADER_TIMEOUT must be greater than zero",
		},
		{
			name:      "zero HTTP write timeout",
			env:       map[string]string{"HTTP_WRITE_TIMEOUT": "0s"},
			wantError: "HTTP_WRITE_TIMEOUT must be greater than zero",
		},
		{
			name:      "zero HTTP idle timeout",
			env:       map[string]string{"HTTP_IDLE_TIMEOUT": "0s"},
			wantError: "HTTP_IDLE_TIMEOUT must be greater than zero",
		},
		{
			name:      "invalid HTTP max body bytes",
			env:       map[string]string{"HTTP_MAX_BODY_BYTES": "many"},
			wantError: "HTTP_MAX_BODY_BYTES must be an integer",
		},
		{
			name:      "zero HTTP max body bytes",
			env:       map[string]string{"HTTP_MAX_BODY_BYTES": "0"},
			wantError: "HTTP_MAX_BODY_BYTES must be greater than zero",
		},
		{
			name:      "invalid log level",
			env:       map[string]string{"LOG_LEVEL": "trace"},
			wantError: "LOG_LEVEL must be one of",
		},
		{
			name:      "invalid shutdown timeout",
			env:       map[string]string{"SHUTDOWN_TIMEOUT": "soon"},
			wantError: "SHUTDOWN_TIMEOUT must be a valid duration",
		},
		{
			name:      "zero shutdown timeout",
			env:       map[string]string{"SHUTDOWN_TIMEOUT": "0s"},
			wantError: "SHUTDOWN_TIMEOUT must be greater than zero",
		},
		{
			name:      "invalid database scheme",
			env:       map[string]string{"DATABASE_URL": "mysql://localhost:3306/app"},
			wantError: "DATABASE_URL must use postgres or postgresql scheme",
		},
		{
			name:      "invalid database connect timeout",
			env:       map[string]string{"DATABASE_CONNECT_TIMEOUT": "soon"},
			wantError: "DATABASE_CONNECT_TIMEOUT must be a valid duration",
		},
		{
			name:      "zero database connect timeout",
			env:       map[string]string{"DATABASE_CONNECT_TIMEOUT": "0s"},
			wantError: "DATABASE_CONNECT_TIMEOUT must be greater than zero",
		},
		{
			name:      "invalid database readiness timeout",
			env:       map[string]string{"DATABASE_READINESS_TIMEOUT": "soon"},
			wantError: "DATABASE_READINESS_TIMEOUT must be a valid duration",
		},
		{
			name:      "zero database readiness timeout",
			env:       map[string]string{"DATABASE_READINESS_TIMEOUT": "0s"},
			wantError: "DATABASE_READINESS_TIMEOUT must be greater than zero",
		},
		{
			name:      "invalid tracing enabled",
			env:       map[string]string{"TRACING_ENABLED": "maybe"},
			wantError: "TRACING_ENABLED must be a boolean",
		},
		{
			name:      "invalid tracing exporter",
			env:       map[string]string{"TRACING_EXPORTER": "otlp"},
			wantError: "TRACING_EXPORTER must be one of",
		},
		{
			name:      "invalid tracing sample ratio",
			env:       map[string]string{"TRACING_SAMPLE_RATIO": "often"},
			wantError: "TRACING_SAMPLE_RATIO must be a number",
		},
		{
			name:      "negative tracing sample ratio",
			env:       map[string]string{"TRACING_SAMPLE_RATIO": "-0.1"},
			wantError: "TRACING_SAMPLE_RATIO must be between 0 and 1",
		},
		{
			name:      "tracing sample ratio above one",
			env:       map[string]string{"TRACING_SAMPLE_RATIO": "1.1"},
			wantError: "TRACING_SAMPLE_RATIO must be between 0 and 1",
		},
		{
			name:      "database URL without host",
			env:       map[string]string{"DATABASE_URL": "postgres:///app"},
			wantError: "DATABASE_URL must include a host",
		},
		{
			name:      "production without database URL",
			env:       map[string]string{"APP_ENV": "production"},
			wantError: "DATABASE_URL is required when APP_ENV is production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := load(mapLookup(tt.env))
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

func mapLookup(values map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

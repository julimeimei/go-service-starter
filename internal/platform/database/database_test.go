package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOpenRejectsNilContext(t *testing.T) {
	t.Parallel()

	_, err := Open(nil, Options{ //nolint:staticcheck // This test verifies defensive nil context handling.
		URL:            "postgres://localhost:5432/app",
		ConnectTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected nil context error, got nil")
	}
}

func TestValidateOptionsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		options   Options
		wantError string
	}{
		{
			name: "missing URL",
			options: Options{
				ConnectTimeout: time.Second,
			},
			wantError: "database URL is required",
		},
		{
			name: "invalid timeout",
			options: Options{
				URL:            "postgres://localhost:5432/app",
				ConnectTimeout: 0,
			},
			wantError: "database connect timeout must be greater than zero",
		},
		{
			name: "invalid scheme",
			options: Options{
				URL:            "mysql://localhost:3306/app",
				ConnectTimeout: time.Second,
			},
			wantError: "database URL must use postgres or postgresql scheme",
		},
		{
			name: "missing host",
			options: Options{
				URL:            "postgres:///app",
				ConnectTimeout: time.Second,
			},
			wantError: "database URL must include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateOptions(tt.options)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

func TestValidateOptionsDoesNotLeakCredentialsOnInvalidURL(t *testing.T) {
	t.Parallel()

	err := validateOptions(Options{
		URL:            "postgres://user:super-secret-password@",
		ConnectTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if strings.Contains(err.Error(), "super-secret-password") {
		t.Fatalf("expected error not to leak credentials, got %q", err.Error())
	}
}

func TestNewReadinessCheckSucceeds(t *testing.T) {
	t.Parallel()

	called := false
	check := NewReadinessCheck(pingFunc(func(ctx context.Context) error {
		called = true

		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("expected readiness check context to have a deadline")
		}

		return nil
	}), time.Second)

	if err := check(context.Background()); err != nil {
		t.Fatalf("expected readiness check to succeed, got %v", err)
	}

	if !called {
		t.Fatal("expected readiness check to ping database")
	}
}

func TestNewReadinessCheckRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		check     func(context.Context) error
		ctx       context.Context
		wantError string
	}{
		{
			name:      "nil context",
			check:     NewReadinessCheck(pingFunc(func(context.Context) error { return nil }), time.Second),
			ctx:       nil,
			wantError: "database readiness context is required",
		},
		{
			name:      "nil pool",
			check:     NewReadinessCheck(nil, time.Second),
			ctx:       context.Background(),
			wantError: "database is not configured",
		},
		{
			name:      "invalid timeout",
			check:     NewReadinessCheck(pingFunc(func(context.Context) error { return nil }), 0),
			ctx:       context.Background(),
			wantError: "database readiness timeout must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.check(tt.ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}

func TestNewReadinessCheckDoesNotLeakPingErrorDetails(t *testing.T) {
	t.Parallel()

	check := NewReadinessCheck(pingFunc(func(context.Context) error {
		return errors.New("password=super-secret-password")
	}), time.Second)

	err := check(context.Background())
	if err == nil {
		t.Fatal("expected readiness check to fail, got nil")
	}

	if err.Error() != "database is not ready" {
		t.Fatalf("expected generic readiness error, got %q", err.Error())
	}

	if strings.Contains(err.Error(), "super-secret-password") {
		t.Fatalf("expected readiness error not to leak credentials, got %q", err.Error())
	}
}

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

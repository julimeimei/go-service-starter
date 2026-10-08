package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	validOptions := Options{
		Addr:              "127.0.0.1:0",
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}

	tests := []struct {
		name    string
		options Options
	}{
		{
			name: "missing address",
			options: func() Options {
				options := validOptions
				options.Addr = ""
				return options
			}(),
		},
		{
			name: "missing handler",
			options: func() Options {
				options := validOptions
				options.Handler = nil
				return options
			}(),
		},
		{
			name: "invalid read header timeout",
			options: func() Options {
				options := validOptions
				options.ReadHeaderTimeout = 0
				return options
			}(),
		},
		{
			name: "invalid read timeout",
			options: func() Options {
				options := validOptions
				options.ReadTimeout = 0
				return options
			}(),
		},
		{
			name: "invalid write timeout",
			options: func() Options {
				options := validOptions
				options.WriteTimeout = 0
				return options
			}(),
		},
		{
			name: "invalid idle timeout",
			options: func() Options {
				options := validOptions
				options.IdleTimeout = 0
				return options
			}(),
		},
		{
			name: "invalid shutdown timeout",
			options: func() Options {
				options := validOptions
				options.ShutdownTimeout = 0
				return options
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(tt.options); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestRunShutsDownWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	srv, err := New(Options{
		Addr: listener.Addr().String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.run(ctx, listener)
	}()

	waitForServer(t, listener.Addr().String())

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected clean shutdown, got error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()

	client := http.Client{Timeout: 100 * time.Millisecond}
	url := fmt.Sprintf("http://%s", addr)
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("failed to create readiness request: %v", err)
		}

		response, err := client.Do(request)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Fatalf("failed to close response body: %v", closeErr)
			}

			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("server did not start before deadline")
}

package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type Options struct {
	Addr              string
	Handler           http.Handler
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	ErrorLog          *log.Logger
}

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func New(options Options) (*Server, error) {
	if options.Addr == "" {
		return nil, fmt.Errorf("server address is required")
	}

	if options.Handler == nil {
		return nil, fmt.Errorf("server handler is required")
	}

	if options.ReadHeaderTimeout <= 0 {
		return nil, fmt.Errorf("server read header timeout must be greater than zero")
	}

	if options.ReadTimeout <= 0 {
		return nil, fmt.Errorf("server read timeout must be greater than zero")
	}

	if options.WriteTimeout <= 0 {
		return nil, fmt.Errorf("server write timeout must be greater than zero")
	}

	if options.IdleTimeout <= 0 {
		return nil, fmt.Errorf("server idle timeout must be greater than zero")
	}

	if options.ShutdownTimeout <= 0 {
		return nil, fmt.Errorf("server shutdown timeout must be greater than zero")
	}

	httpServer := &http.Server{
		Addr:              options.Addr,
		Handler:           options.Handler,
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		ReadTimeout:       options.ReadTimeout,
		WriteTimeout:      options.WriteTimeout,
		IdleTimeout:       options.IdleTimeout,
		ErrorLog:          options.ErrorLog,
	}

	return &Server{
		httpServer:      httpServer,
		shutdownTimeout: options.ShutdownTimeout,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.run(ctx, nil)
}

func (s *Server) run(ctx context.Context, listener net.Listener) error {
	if ctx == nil {
		return fmt.Errorf("server context is required")
	}

	s.httpServer.BaseContext = func(net.Listener) context.Context {
		return ctx
	}

	errCh := make(chan error, 1)

	go func() {
		if listener != nil {
			errCh <- s.httpServer.Serve(listener)
			return
		}

		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return normalizeServerError(err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		return nil
	}
}

func normalizeServerError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("server listen failed: %w", err)
}

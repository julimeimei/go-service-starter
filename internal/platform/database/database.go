package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	URL            string
	ConnectTimeout time.Duration
}

type Pinger interface {
	Ping(context.Context) error
}

func Open(ctx context.Context, options Options) (*pgxpool.Pool, error) {
	if ctx == nil {
		return nil, fmt.Errorf("database context is required")
	}

	if err := validateOptions(options); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(options.URL)
	if err != nil {
		return nil, fmt.Errorf("database URL is invalid")
	}

	connectCtx, cancel := context.WithTimeout(ctx, options.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("database pool creation failed")
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed")
	}

	return pool, nil
}

func NewReadinessCheck(pool Pinger, timeout time.Duration) func(context.Context) error {
	return func(ctx context.Context) error {
		if ctx == nil {
			return fmt.Errorf("database readiness context is required")
		}

		if pool == nil {
			return fmt.Errorf("database is not configured")
		}

		if timeout <= 0 {
			return fmt.Errorf("database readiness timeout must be greater than zero")
		}

		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := pool.Ping(checkCtx); err != nil {
			return fmt.Errorf("database is not ready")
		}

		return nil
	}
}

func validateOptions(options Options) error {
	if options.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if options.ConnectTimeout <= 0 {
		return fmt.Errorf("database connect timeout must be greater than zero")
	}

	parsed, err := url.Parse(options.URL)
	if err != nil {
		return fmt.Errorf("database URL is invalid")
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("database URL must use postgres or postgresql scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("database URL must include a host")
	}

	return nil
}

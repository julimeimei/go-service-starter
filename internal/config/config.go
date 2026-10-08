package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAppName               = "go-service-starter"
	defaultAppEnv                = "development"
	defaultHTTPHost              = "127.0.0.1"
	defaultHTTPPort              = 8080
	defaultHTTPReadTimeout       = 10 * time.Second
	defaultHTTPReadHeaderTimeout = 5 * time.Second
	defaultHTTPWriteTimeout      = 10 * time.Second
	defaultHTTPIdleTimeout       = 60 * time.Second
	defaultHTTPMaxBodyBytes      = 1 << 20
	defaultDatabaseConnectTime   = 5 * time.Second
	defaultDatabaseReadinessTime = 2 * time.Second
	defaultLogLevel              = "info"
	defaultShutdownTimeout       = 10 * time.Second
	defaultTracingEnabled        = false
	defaultTracingExporter       = "stdout"
	defaultTracingSampleRatio    = 1.0
)

type Config struct {
	App             AppConfig
	HTTP            HTTPConfig
	Log             LogConfig
	Database        DatabaseConfig
	Tracing         TracingConfig
	ShutdownTimeout time.Duration
}

type AppConfig struct {
	Name string
	Env  string
}

type HTTPConfig struct {
	Host              string
	Port              int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxBodyBytes      int64
}

func (c HTTPConfig) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

type LogConfig struct {
	Level string
}

type DatabaseConfig struct {
	URL              string
	ConnectTimeout   time.Duration
	ReadinessTimeout time.Duration
}

type TracingConfig struct {
	Enabled     bool
	Exporter    string
	SampleRatio float64
}

type lookupFunc func(string) (string, bool)

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup lookupFunc) (Config, error) {
	httpPort, err := readInt(lookup, "HTTP_PORT", defaultHTTPPort)
	if err != nil {
		return Config{}, err
	}

	httpReadTimeout, err := readDuration(lookup, "HTTP_READ_TIMEOUT", defaultHTTPReadTimeout)
	if err != nil {
		return Config{}, err
	}

	httpReadHeaderTimeout, err := readDuration(lookup, "HTTP_READ_HEADER_TIMEOUT", defaultHTTPReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	httpWriteTimeout, err := readDuration(lookup, "HTTP_WRITE_TIMEOUT", defaultHTTPWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	httpIdleTimeout, err := readDuration(lookup, "HTTP_IDLE_TIMEOUT", defaultHTTPIdleTimeout)
	if err != nil {
		return Config{}, err
	}

	httpMaxBodyBytes, err := readInt64(lookup, "HTTP_MAX_BODY_BYTES", defaultHTTPMaxBodyBytes)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := readDuration(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	databaseConnectTimeout, err := readDuration(lookup, "DATABASE_CONNECT_TIMEOUT", defaultDatabaseConnectTime)
	if err != nil {
		return Config{}, err
	}

	databaseReadinessTimeout, err := readDuration(lookup, "DATABASE_READINESS_TIMEOUT", defaultDatabaseReadinessTime)
	if err != nil {
		return Config{}, err
	}

	tracingEnabled, err := readBool(lookup, "TRACING_ENABLED", defaultTracingEnabled)
	if err != nil {
		return Config{}, err
	}

	tracingSampleRatio, err := readFloat64(lookup, "TRACING_SAMPLE_RATIO", defaultTracingSampleRatio)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Name: readString(lookup, "APP_NAME", defaultAppName),
			Env:  readString(lookup, "APP_ENV", defaultAppEnv),
		},
		HTTP: HTTPConfig{
			Host:              readString(lookup, "HTTP_HOST", defaultHTTPHost),
			Port:              httpPort,
			ReadTimeout:       httpReadTimeout,
			ReadHeaderTimeout: httpReadHeaderTimeout,
			WriteTimeout:      httpWriteTimeout,
			IdleTimeout:       httpIdleTimeout,
			MaxBodyBytes:      httpMaxBodyBytes,
		},
		Log: LogConfig{
			Level: readString(lookup, "LOG_LEVEL", defaultLogLevel),
		},
		Database: DatabaseConfig{
			URL:              readString(lookup, "DATABASE_URL", ""),
			ConnectTimeout:   databaseConnectTimeout,
			ReadinessTimeout: databaseReadinessTimeout,
		},
		Tracing: TracingConfig{
			Enabled:     tracingEnabled,
			Exporter:    readString(lookup, "TRACING_EXPORTER", defaultTracingExporter),
			SampleRatio: tracingSampleRatio,
		},
		ShutdownTimeout: shutdownTimeout,
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME is required")
	}

	if !isAllowed(c.App.Env, []string{"development", "test", "staging", "production"}) {
		return fmt.Errorf("APP_ENV must be one of: development, test, staging, production")
	}

	if c.HTTP.Host == "" {
		return fmt.Errorf("HTTP_HOST is required")
	}

	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535")
	}

	if c.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_TIMEOUT must be greater than zero")
	}

	if c.HTTP.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("HTTP_READ_HEADER_TIMEOUT must be greater than zero")
	}

	if c.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("HTTP_WRITE_TIMEOUT must be greater than zero")
	}

	if c.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("HTTP_IDLE_TIMEOUT must be greater than zero")
	}

	if c.HTTP.MaxBodyBytes <= 0 {
		return fmt.Errorf("HTTP_MAX_BODY_BYTES must be greater than zero")
	}

	if !isAllowed(c.Log.Level, []string{"debug", "info", "warn", "error"}) {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be greater than zero")
	}

	if c.Database.ConnectTimeout <= 0 {
		return fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be greater than zero")
	}

	if c.Database.ReadinessTimeout <= 0 {
		return fmt.Errorf("DATABASE_READINESS_TIMEOUT must be greater than zero")
	}

	if !isAllowed(c.Tracing.Exporter, []string{"stdout"}) {
		return fmt.Errorf("TRACING_EXPORTER must be one of: stdout")
	}

	if c.Tracing.SampleRatio < 0 || c.Tracing.SampleRatio > 1 {
		return fmt.Errorf("TRACING_SAMPLE_RATIO must be between 0 and 1")
	}

	if c.Database.URL == "" {
		if c.App.Env == "production" {
			return fmt.Errorf("DATABASE_URL is required when APP_ENV is production")
		}

		return nil
	}

	parsed, err := url.Parse(c.Database.URL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid")
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use postgres or postgresql scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("DATABASE_URL must include a host")
	}

	return nil
}

func readString(lookup lookupFunc, key, fallback string) string {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}

	return strings.TrimSpace(value)
}

func readInt(lookup lookupFunc, key string, fallback int) (int, error) {
	raw := readString(lookup, key, "")
	if raw == "" {
		if _, ok := lookup(key); ok {
			return 0, fmt.Errorf("%s is required", key)
		}

		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return value, nil
}

func readInt64(lookup lookupFunc, key string, fallback int64) (int64, error) {
	raw := readString(lookup, key, "")
	if raw == "" {
		if _, ok := lookup(key); ok {
			return 0, fmt.Errorf("%s is required", key)
		}

		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return value, nil
}

func readDuration(lookup lookupFunc, key string, fallback time.Duration) (time.Duration, error) {
	raw := readString(lookup, key, "")
	if raw == "" {
		if _, ok := lookup(key); ok {
			return 0, fmt.Errorf("%s is required", key)
		}

		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}

	return value, nil
}

func readBool(lookup lookupFunc, key string, fallback bool) (bool, error) {
	raw := readString(lookup, key, "")
	if raw == "" {
		if _, ok := lookup(key); ok {
			return false, fmt.Errorf("%s is required", key)
		}

		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}

	return value, nil
}

func readFloat64(lookup lookupFunc, key string, fallback float64) (float64, error) {
	raw := readString(lookup, key, "")
	if raw == "" {
		if _, ok := lookup(key); ok {
			return 0, fmt.Errorf("%s is required", key)
		}

		return fallback, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}

	return value, nil
}

func isAllowed(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}

	return false
}

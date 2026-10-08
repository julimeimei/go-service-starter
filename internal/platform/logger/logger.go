package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Options struct {
	Level  string
	Env    string
	Output io.Writer
}

func New(options Options) (*slog.Logger, error) {
	level, err := parseLevel(options.Level)
	if err != nil {
		return nil, err
	}

	output := options.Output
	if output == nil {
		output = os.Stdout
	}

	handlerOptions := &slog.HandlerOptions{
		Level: level,
	}

	if options.Env == "production" {
		return slog.New(slog.NewJSONHandler(output, handlerOptions)), nil
	}

	return slog.New(slog.NewTextHandler(output, handlerOptions)), nil
}

func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", level)
	}
}

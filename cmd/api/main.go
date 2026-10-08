package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/julimeimei/go-service-starter/internal/config"
	httpapi "github.com/julimeimei/go-service-starter/internal/http"
	"github.com/julimeimei/go-service-starter/internal/items"
	"github.com/julimeimei/go-service-starter/internal/platform/database"
	"github.com/julimeimei/go-service-starter/internal/platform/logger"
	"github.com/julimeimei/go-service-starter/internal/platform/metrics"
	"github.com/julimeimei/go-service-starter/internal/platform/observability"
	"github.com/julimeimei/go-service-starter/internal/platform/server"
	"github.com/julimeimei/go-service-starter/internal/platform/tracing"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(logger.Options{
		Level: cfg.Log.Level,
		Env:   cfg.App.Env,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	traceShutdown, err := tracing.Setup(ctx, tracing.Options{
		Enabled:     cfg.Tracing.Enabled,
		ServiceName: cfg.App.Name,
		Environment: cfg.App.Env,
		Exporter:    cfg.Tracing.Exporter,
		SampleRatio: cfg.Tracing.SampleRatio,
	})
	if err != nil {
		log.Error("tracing configuration error", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := tracing.ShutdownContext(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := traceShutdown(shutdownCtx); err != nil {
			log.Warn("tracing shutdown failed", "error", err)
		}
	}()

	var readinessCheck httpapi.ReadinessCheck
	var itemsHandler http.Handler
	if cfg.Database.URL != "" {
		dbpool, err := database.Open(ctx, database.Options{
			URL:            cfg.Database.URL,
			ConnectTimeout: cfg.Database.ConnectTimeout,
		})
		if err != nil {
			log.Error("database connection failed", "error", err)
			os.Exit(1)
		}
		defer dbpool.Close()

		log.Info("database connected")
		readinessCheck = database.NewReadinessCheck(dbpool, cfg.Database.ReadinessTimeout)

		itemsRepository := items.NewPostgresRepository(dbpool)
		itemsService := items.NewService(itemsRepository)
		itemsHandler = items.NewHandler(itemsService, log)
	} else {
		log.Warn("database not configured; skipping connection")
	}

	metricsRegistry := metrics.NewRegistry()
	httpMetrics, err := metrics.NewHTTPMetrics(metricsRegistry)
	if err != nil {
		log.Error("metrics configuration error", "error", err)
		os.Exit(1)
	}

	router := httpapi.NewRouter(httpapi.RouterConfig{
		ServiceName:       cfg.App.Name,
		MaxBodyBytes:      cfg.HTTP.MaxBodyBytes,
		Logger:            log,
		ReadinessCheck:    readinessCheck,
		ItemsHandler:      itemsHandler,
		MetricsHandler:    metrics.Handler(metricsRegistry),
		MetricsMiddleware: httpMetrics.Middleware,
		TracingMiddleware: tracing.HTTPMiddleware(tracing.HTTPMiddlewareOptions{
			ServiceName:  cfg.App.Name,
			RoutePattern: observability.RoutePattern,
		}),
	})

	apiServer, err := server.New(server.Options{
		Addr:              cfg.HTTP.Addr(),
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		ShutdownTimeout:   cfg.ShutdownTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	})
	if err != nil {
		log.Error("server configuration error", "error", err)
		os.Exit(1)
	}

	log.Info(
		"server starting",
		"app_name", cfg.App.Name,
		"app_env", cfg.App.Env,
		"addr", cfg.HTTP.Addr(),
	)

	if err := apiServer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}

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
	"time"

	"github.com/LilianMrt/go-listings-service/internal/config"
	"github.com/LilianMrt/go-listings-service/internal/db"
	"github.com/LilianMrt/go-listings-service/internal/events"
	"github.com/LilianMrt/go-listings-service/internal/httpapi"
	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/listing/store"
	"github.com/LilianMrt/go-listings-service/internal/observability"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("service exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Apply schema migrations before opening the application pool.
	logger.Info("applying database migrations")
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Choose the event publisher: Kafka when enabled, otherwise a noop that
	// discards events so the service still runs without a broker.
	var publisher listing.Publisher = events.Noop{}
	if cfg.KafkaEnabled {
		kw := events.NewWriter(cfg.KafkaBrokers, cfg.KafkaTopic)
		defer func() {
			if err := kw.Close(); err != nil {
				logger.Warn("closing kafka writer", slog.Any("error", err))
			}
		}()
		publisher = kw
		logger.Info("kafka publisher enabled",
			slog.String("topic", cfg.KafkaTopic),
			slog.Any("brokers", cfg.KafkaBrokers))
	} else {
		logger.Info("kafka disabled, listing events will be discarded")
	}

	repo := store.NewPostgres(pool)
	svc := listing.NewService(repo, listing.WithPublisher(publisher))

	health := observability.NewHealth()

	router := httpapi.NewRouter(httpapi.Deps{
		Logger:   logger,
		Health:   health,
		Listings: svc,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Migrations ran and the pool pinged, so dependencies are reachable.
	health.SetReady(true)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
		health.SetReady(false)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.Info("shutdown complete")
		return nil
	}
}

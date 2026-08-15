package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		logger.Error("invalid configuration", "error", "DATABASE_URL is required")
		os.Exit(1)
	}
	database, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("create database pool", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err = database.Ping(ctx); err != nil {
		logger.Error("database is unavailable", "error", err)
		os.Exit(1)
	}

	worker := notifications.NewWorker(database)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	logger.Info("worker started", "processor", "notification-outbox")
	for {
		processed, processErr := worker.ProcessBatch(ctx, 50)
		if processErr != nil && ctx.Err() == nil {
			logger.Error("outbox batch failed", "error", processErr)
		} else if processed > 0 {
			logger.Info("outbox batch processed", "count", processed)
		}
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return
		case <-ticker.C:
		}
	}
}

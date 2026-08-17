package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dx-os-lab/dx-os/backend/internal/notifications"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/config"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/documentstore"
	"github.com/dx-os-lab/dx-os/backend/internal/platform/httpapi"
	"github.com/dx-os-lab/dx-os/backend/internal/procurement"
	"github.com/dx-os-lab/dx-os/backend/internal/reporting"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := pgxpool.New(rootContext, cfg.DatabaseURL)
	if err != nil {
		logger.Error("create database pool", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	pingContext, cancelPing := context.WithTimeout(rootContext, 5*time.Second)
	err = database.Ping(pingContext)
	cancelPing()
	if err != nil {
		logger.Error("database is unavailable", "error", err)
		os.Exit(1)
	}

	tokenVerifier, err := auth.NewVerifier(rootContext, auth.VerifierConfig{
		Issuer:   cfg.OIDCIssuer,
		Audience: cfg.OIDCAudience,
		JWKSURL:  cfg.OIDCJWKSURL,
	})
	if err != nil {
		logger.Error("initialize OIDC verifier", "error", err)
		os.Exit(1)
	}
	documents := documentstore.NewNextcloud(
		cfg.NextcloudURL,
		cfg.NextcloudUsername,
		cfg.NextcloudPassword,
		cfg.NextcloudRoot,
	)

	procurementStore := procurement.NewStore(database, documents)
	handler := httpapi.New(httpapi.Dependencies{
		AllowedOrigin: cfg.AllowedOrigin,
		Database:      database,
		Logger:        logger,
		Notifications: notifications.NewStore(database),
		Procurement:   procurementStore,
		Enterprise:    procurementStore,
		Reporting:     reporting.NewStore(database),
		TokenVerifier: tokenVerifier,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", cfg.HTTPAddress, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-rootContext.Done():
		logger.Info("shutdown requested")
	case err = <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err = server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}

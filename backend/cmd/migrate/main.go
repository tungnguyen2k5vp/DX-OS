package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	database, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fail(logger, "create database pool", err)
	}
	defer database.Close()

	if err = run(ctx, database, migrationsPath, logger); err != nil {
		fail(logger, "run migrations", err)
	}
}

type database interface {
	Exec(context.Context, string, ...any) (any, error)
}

func run(ctx context.Context, pool *pgxpool.Pool, migrationsPath string, logger *slog.Logger) error {
	migrationsRoot, err := os.OpenRoot(migrationsPath)
	if err != nil {
		return fmt.Errorf("open migrations directory: %w", err)
	}
	defer migrationsRoot.Close()
	entries, err := fs.ReadDir(migrationsRoot.FS(), ".")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	if _, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			checksum_sha256 char(64),
			applied_at timestamptz NOT NULL DEFAULT now()
		);
		ALTER TABLE schema_migrations
			ADD COLUMN IF NOT EXISTS checksum_sha256 char(64)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		contents, readErr := migrationsRoot.ReadFile(name)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}
		checksum := migrationChecksum(contents)

		var recordedChecksum *string
		err = pool.QueryRow(ctx,
			"SELECT checksum_sha256 FROM schema_migrations WHERE version = $1",
			name,
		).Scan(&recordedChecksum)
		switch {
		case err == nil:
			if recordedChecksum == nil || strings.TrimSpace(*recordedChecksum) == "" {
				if _, err = pool.Exec(ctx,
					"UPDATE schema_migrations SET checksum_sha256 = $2 WHERE version = $1",
					name, checksum,
				); err != nil {
					return fmt.Errorf("backfill migration checksum %s: %w", name, err)
				}
				continue
			}
			recorded := strings.TrimSpace(*recordedChecksum)
			if recorded != checksum {
				// Early DX-OS databases stored checksums from Windows CRLF files.
				// Treat only this byte-level line-ending variation as compatible;
				// any substantive migration edit remains blocked.
				if recorded == legacyWindowsLineEndingChecksum(contents) {
					if _, err = pool.Exec(ctx,
						"UPDATE schema_migrations SET checksum_sha256 = $2 WHERE version = $1",
						name, checksum,
					); err != nil {
						return fmt.Errorf("normalize migration checksum %s: %w", name, err)
					}
					logger.Info("normalized legacy migration checksum", "version", name)
					continue
				}
				return fmt.Errorf("migration %s checksum mismatch: an applied migration was modified", name)
			}
			continue
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		tx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			return fmt.Errorf("begin migration %s: %w", name, beginErr)
		}
		if _, err = tx.Exec(ctx, string(contents)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, checksum_sha256) VALUES ($1, $2)",
			name, checksum,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
		logger.Info("migration applied", "version", name)
	}

	logger.Info("database migrations are current", "count", len(files))
	return nil
}

func migrationChecksum(contents []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(normalizeLineEndings(contents)))
}

func legacyWindowsLineEndingChecksum(contents []byte) string {
	legacyContents := bytes.ReplaceAll(normalizeLineEndings(contents), []byte("\n"), []byte("\r\n"))
	return fmt.Sprintf("%x", sha256.Sum256(legacyContents))
}

func normalizeLineEndings(contents []byte) []byte {
	return bytes.ReplaceAll(contents, []byte("\r\n"), []byte("\n"))
}

func fail(logger *slog.Logger, message string, err error) {
	logger.Error(message, "error", err)
	os.Exit(1)
}

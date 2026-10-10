package main

import (
	"context"
	"log/slog"
	"os"

	"selfemi/internal/platform/config"
	"selfemi/internal/platform/database"
	"selfemi/internal/platform/migrate"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := migrate.Apply(ctx, pool, os.DirFS(cfg.MigrationsDir))
	if err != nil {
		return err
	}
	logger.Info("migrations complete", "applied", len(applied), "files", applied)
	return nil
}

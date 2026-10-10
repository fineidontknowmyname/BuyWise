package config

import "errors"

const (
	DefaultAPIAddr       = "127.0.0.1:8080"
	DefaultMigrationsDir = "migrations"
)

var ErrMissingDatabaseURL = errors.New("DATABASE_URL is not set")

type Config struct {
	APIAddr       string
	DatabaseURL   string
	MigrationsDir string
}

func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		APIAddr:       valueOr(getenv("API_ADDR"), DefaultAPIAddr),
		DatabaseURL:   getenv("DATABASE_URL"),
		MigrationsDir: valueOr(getenv("MIGRATIONS_DIR"), DefaultMigrationsDir),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, ErrMissingDatabaseURL
	}
	return cfg, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

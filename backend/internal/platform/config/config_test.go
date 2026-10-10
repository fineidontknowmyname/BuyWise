package config

import (
	"errors"
	"testing"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := Load(envFrom(map[string]string{"DATABASE_URL": "postgres://example"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIAddr != DefaultAPIAddr {
		t.Errorf("APIAddr = %q, want %q", cfg.APIAddr, DefaultAPIAddr)
	}
	if cfg.MigrationsDir != DefaultMigrationsDir {
		t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, DefaultMigrationsDir)
	}
}

func TestLoadUsesProvidedValues(t *testing.T) {
	cfg, err := Load(envFrom(map[string]string{
		"DATABASE_URL":   "postgres://example",
		"API_ADDR":       "127.0.0.1:9999",
		"MIGRATIONS_DIR": "other",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIAddr != "127.0.0.1:9999" || cfg.MigrationsDir != "other" {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := Load(envFrom(map[string]string{}))
	if !errors.Is(err, ErrMissingDatabaseURL) {
		t.Fatalf("err = %v, want ErrMissingDatabaseURL", err)
	}
}

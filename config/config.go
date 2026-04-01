package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	StorageMemory   = "memory"
	StoragePostgres = "postgres"
)

var ErrUnsupportedStorage = errors.New("unsupported storage type")

type Config struct {
	HTTPAddr string
	BaseURL  string
	Storage  string
	Postgres PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func Load() (Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cfg := Config{}
	fs.StringVar(&cfg.HTTPAddr, "addr", envOrDefault("APP_ADDR", ":8080"), "http listen address")
	fs.StringVar(&cfg.BaseURL, "base-url", envOrDefault("BASE_URL", "http://localhost:8080"), "public base url")
	fs.StringVar(&cfg.Storage, "storage", envOrDefault("STORAGE_TYPE", StorageMemory), "storage type: memory or postgres")

	fs.StringVar(&cfg.Postgres.Host, "db-host", envOrDefault("DB_HOST", "localhost"), "postgres host")
	fs.StringVar(&cfg.Postgres.Port, "db-port", envOrDefault("DB_PORT", "5432"), "postgres port")
	fs.StringVar(&cfg.Postgres.User, "db-user", envOrDefault("DB_USER", "postgres"), "postgres user")
	fs.StringVar(&cfg.Postgres.Password, "db-password", envOrDefault("DB_PASSWORD", "postgres"), "postgres password")
	fs.StringVar(&cfg.Postgres.DBName, "db-name", envOrDefault("DB_NAME", "url_shortener"), "postgres database name")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Storage {
	case StorageMemory, StoragePostgres:
		return nil
	default:
		return fmt.Errorf("%w %q", ErrUnsupportedStorage, c.Storage)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

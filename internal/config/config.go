// Package config loads and validates trip service configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains application settings loaded from environment variables.
type Config struct {
	HTTPAddr                string
	HTTPReadTimeout         time.Duration
	HTTPReadHeaderTimeout   time.Duration
	HTTPWriteTimeout        time.Duration
	HTTPIdleTimeout         time.Duration
	DatabaseURL             string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration
}

// Load reads and validates the required service configuration from the environment.
func Load() (Config, error) {
	var cfg Config
	var err error

	if cfg.HTTPAddr, err = requiredString("HTTP_ADDR"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseURL, err = requiredString("DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadTimeout, err = durationOrDefault("HTTP_READ_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadHeaderTimeout, err = durationOrDefault("HTTP_READ_HEADER_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTPWriteTimeout, err = durationOrDefault("HTTP_WRITE_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTPIdleTimeout, err = durationOrDefault("HTTP_IDLE_TIMEOUT", time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.LogLevel, err = requiredString("LOG_LEVEL"); err != nil {
		return Config{}, err
	}
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error; got %q", cfg.LogLevel)
	}

	if cfg.ShutdownTimeout, err = requiredDuration("SHUTDOWN_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseConnectTimeout, err = requiredDuration("DATABASE_CONNECT_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseQueryTimeout, err = requiredDuration("DATABASE_QUERY_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConnLifetime, err = requiredDuration("DATABASE_MAX_CONN_LIFETIME"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns, err = requiredInt32("DATABASE_MAX_CONNS"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns, err = requiredInt32("DATABASE_MIN_CONNS"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns < 1 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must be between zero and DATABASE_MAX_CONNS")
	}

	return cfg, nil
}

func requiredString(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

func requiredDuration(name string) (time.Duration, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return duration, nil
}

func durationOrDefault(name string, fallback time.Duration) (time.Duration, error) {
	if strings.TrimSpace(os.Getenv(name)) == "" {
		return fallback, nil
	}
	return requiredDuration(name)
}

func requiredInt32(name string) (int32, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s as integer: %w", name, err)
	}
	return int32(parsed), nil
}

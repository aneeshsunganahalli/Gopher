package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server ServerConfig `envconfig:"SERVER"`
	Redis  RedisConfig  `envconfig:"REDIS"`
	Worker WorkerConfig `envconfig:"WORKER"`
	Log    LogConfig    `envconfig:"LOG"`
}

type ServerConfig struct {
	Port         int           `envconfig:"PORT" default:"8080"`
	Host         string        `envconfig:"HOST" default:"localhost"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
}

type RedisConfig struct {
	URL      string        `envconfig:"URL" default:"redis://localhost:6379"`
	Password string        `envconfig:"PASSWORD" default:""`
	DB       int           `envconfig:"DB" default:"0"`
	Timeout  time.Duration `envconfig:"TIMEOUT" default:"5s"`
}

type WorkerConfig struct {
	Concurrency     int           `envconfig:"CONCURRENCY" default:"5"`
	PollInterval    time.Duration `envconfig:"POLL_INTERVAL" default:"1s"`
	MaxRetries      int           `envconfig:"MAX_RETRIES" default:"3"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"30s"`
}

type LogConfig struct {
	Level  string `envconfig:"LEVEL"  default:"info"`
	Format string `envconfig:"FORMAT" default:"console"` // json in prod
}

// Address returns the full server address
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Load reads config from env variables
func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("Config validation failed: %w", err)
	}

	return &cfg, nil
}

// Config Validator
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Worker.Concurrency <= 0 {
		return fmt.Errorf("worker Concurrency must be positive, got: %d", c.Worker.Concurrency)
	}

	if c.Worker.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative, got: %d", c.Worker.MaxRetries)
	}

	return nil
}
